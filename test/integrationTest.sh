#!/bin/bash

# Required environment variables for Azure
# AZURE_TENANT_ID
# AZURE_CLIENT_ID
# AZURE_CLIENT_SECRET

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

# stops the test
# first argument: exit code the program should have
# second argument: the message it should print
stopTest(){
    sudo ./teardownDB.sh 2>&1 | while read line; do echo -e "\e[95mdatabase:\e[0m $line"; done
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
    sudo ./setupDB.sh 2>&1 | while read line; do echo -e "\e[95mdatabase:\e[0m $line"; done
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
        while read line; do echo -e "\e[93m$1:\e[0m $line"; done
    goCode=$?
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


#name: cost
#time			        cloud	cost			currency	provider				service
#----			        -----	----			--------	--------				-------
#1507075200000000000	azure	0.1949901520896		SEK							Microsoft.Compute/disks
#1507075200000000000		    0.6744044159999999	SEK		                    Microsoft.Network/publicIPAddresses
#1507075200000000000	azure	12.4593326968176	SEK							Microsoft.Compute/virtualMachines
#1507075200000000000		    0.1949901520896		SEK		                    Microsoft.Compute/disks
#1507075200000000000		    12.4593326968176	SEK		                    Microsoft.Compute/virtualMachines
#1507075200000000000	azure	0.6744044159999999	SEK							Microsoft.Network/publicIPAddresses
#1507161600000000000		    12.4679808		    SEK		                    Microsoft.Compute/virtualMachines
#1507161600000000000	azure	0.6772380479999999	SEK							Microsoft.Network/publicIPAddresses
#1507161600000000000		    .6772380479999999	SEK		                    Microsoft.Network/publicIPAddresses
#1507161600000000000	azure	0.1949901520896		SEK							Microsoft.Compute/disks
#1507161600000000000		    0.1949901520896		SEK		                    Microsoft.Compute/disks
#1507161600000000000	azure	12.4679808		S   EK							Microsoft.Compute/virtualMachines

#name: cost
#time			cloud	cost			currency	service
#----			-----	----			--------	-------
#1507075200000000000	azure	0.1949901520896		SEK		Microsoft.Compute/disks
#1507075200000000000	azure	0.6744044159999999	SEK		Microsoft.Network/publicIPAddresses
#1507075200000000000	azure	12.4593326968176	SEK		Microsoft.Compute/virtualMachines
#1507161600000000000	azure	0.6772380479999999	SEK		Microsoft.Network/publicIPAddresses
#1507161600000000000	azure	12.4679808		SEK		Microsoft.Compute/virtualMachines
#1507161600000000000	azure	0.1949901520896		SEK		Microsoft.Compute/disks
