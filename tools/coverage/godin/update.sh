#!/bin/bash

# This script periodically publishes coverage reports of dynamically
# executed golang binaries. 

# Open goc server
echo "Starting goc server"
/goc server &>/dev/null &


# Start a lightweight webserver to make coverage report
# available at http://localip:80
# The default path is  /var/www/localhost/htdocs/
echo "Starting lcov webserver"
lighttpd -f /etc/lighttpd/lighttpd.conf

# Start the instrumented binary
echo "Starting the instrumented binary"
/instr_bin &> /var/log/instr_bin.out &


# Give some time for all the previous servers to come up.
echo "Waiting 10 seconds for services to become ready..."
sleep 10

# Periodically update the coverage report
while true
do
 echo "Update coverage"
 # Get the report in the go cover format
 /goc profile  | \
 # Convert it to lcov format
 /gcov2lcov > /tmp/profile.out  | \
 # Generate http. 
 genhtml -o /var/www/localhost/htdocs/  /tmp/profile.out &> /dev/null

 sleep 30
done

