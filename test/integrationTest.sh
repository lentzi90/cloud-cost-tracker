#!/bin/bash

# Assumes that:
#   * aws cli configured
#   * Environment variables for Azure set
#     - AZURE_TENANT_ID
#     - AZURE_CLIENT_ID
#     - AZURE_CLIENT_SECRET
#   * influxdb-client is installed

DATABASEHOST="localhost"
DATABASEPORT=8086
DATABASE="cloudCostTracker"
DATABASEUSER="cctUser"
DATABASEPASSWORD="cctPassword"

cd "$( dirname "${BASH_SOURCE[0]}" )"

logInfo(){
    echo -e "\e[94mInfo:\e[0m $1"
}

logError(){
    echo -e "\e[91mError:\e[0m $1"
}

logSuccess(){
    echo -e "\e[92mSuccess:\e[0m $1"
}

# Prepends a label to each new line with a specific color
# first argument: The color code that the string should have, eg. \e[92m
# second argument: The label to prepend
prependString(){
    { x=1; while IFS= read -d'' -s -N 1 char; do
        [ $x ] && printf "$1$2:\e[0m "
        printf "$char"
        unset x
        [ "$char" = "
" ] && x=1 # Important that no this one starts at the beginning of the line (no tabs/spaces)
    done; }
}

# stops the test
# first argument: exit code the program should have
# second argument: the message it should print
stopTest(){
    sudo ./teardownDB.sh 2>&1 | prependString "\e[95m" "database"
    echo ""
    if [ $1 -eq 0 ]; then
        logSuccess "$2"
    else
        logError "$2"
    fi
    exit $1
}

# Starts the DB and asserts that it's empty
startDBAndCheckIfEmpty(){
    sudo ./setupDB.sh 2>&1 | prependString "\e[95m" "database"
    logInfo "Waiting for database to start"

    # Wait for database to properly start
    databases=""
    i=0
    while [[ ! $databases = *"$DATABASE"* ]]; do
        sleep 0.5
        databases=$(influx -host $DATABASEHOST -port $DATABASEPORT -username $DATABASEUSER -password $DATABASEPASSWORD -database $DATABASE -execute 'SHOW DATABASES' 2>/dev/null)
        i=$(($i+1))
        if [ $i -gt 20 ]; then
            stopTest 1 "Database didn't start in time, is influxdb-client installed"
        fi
    done

    # Empty DB
    dbOut=$(influx -host $DATABASEHOST -port $DATABASEPORT -username $DATABASEUSER -password $DATABASEPASSWORD -database $DATABASE -execute 'SELECT * FROM cost')

    if [[ $dbOut != "" ]]; then
        stopTest 1 "Database not empty after start, maybe it's already running"
    fi
}

# stops the test
# first argument: which cloud provider to connect to
runTest(){
    logInfo "Fetching data for $1"
    # Fetch Data
    go run ../cmd/cct/main.go \
        --cloud $1 \
        --db-address http://$DATABASEHOST:$DATABASEPORT \
        --db-name $DATABASE \
        --db-username $DATABASEUSER --db-password $DATABASEPASSWORD 2>&1 | \
        prependString "\e[93m" "$1"; goCode=${PIPESTATUS[0]}

    logInfo "Program done."
    if [ $goCode -ne 0 ]; then
        stopTest 1 "Program terminated with code $goCode when fetching data for $1"
    fi

    # Non empty DB
    dbOut=$(influx -host $DATABASEHOST -port $DATABASEPORT -username $DATABASEUSER -password $DATABASEPASSWORD -database $DATABASE -execute "SELECT * FROM cost WHERE cloud = '$1'")

    if [[ $dbOut == "" ]]; then
        stopTest 1 "Database empty after filling with $1 data"
    fi

    # Parse outout to be able to check fields
    fields=$(awk 'FNR == 2 {print}' <<< "$dbOut" | sed -e 's/[[:space:]]\+/, /g')
    fields=" $fields "

    if [[ ! $dbOut = *[[:space:]]"cloud"[[:space:],]* ]]; then
        stopTest 1 "The field \"cloud\" in the database aren't set!\nGot:$fields"
    elif [[ ! $dbOut = *[[:space:]]"cost"[[:space:],]* ]]; then
        stopTest 1 "The field \"cost\" in the database aren't set!\nGot:$fields"
    elif [[ ! $dbOut = *[[:space:]]"currency"[[:space:],]* ]]; then
        stopTest 1 "The field \"currency\" in the database aren't set!\nGot:$fields"
    elif [[ ! $dbOut = *[[:space:]]"service"[[:space:],]* ]]; then
        stopTest 1 "The field \"service\" in the database aren't set!\nGot:$fields"
    fi
}

startDBAndCheckIfEmpty

runTest "azure"
logSuccess "Completed test for Azure"
runTest "aws"
logSuccess "Completed test for AWS"

stopTest 0 "Test completed successfully!"
