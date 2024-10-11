#!/usr/bin/env bash

VERBOSE=0

clean () {
    go clean $( [ "${VERBOSE}" = "1" ] && echo "-x" )
    rm -f bin/${project}
}

compile () {
    go build -o bin/${project} $( [ "${VERBOSE}" = "1" ] && echo "-x -v" )
    # TODO: move this to Dockerfile COPY when it's using a specific user
    chmod +x bin/${project}
}

package () {
    APP_NAME="${project}" docker build -f scripts/Dockerfile .
}

build () {
    compile
    package    
}

#start () {
#    stop
#}

#stop () {

#}

# entry point
cd -- $( dirname -- "${BASH_SOURCE[0]}")/..
project="${PWD##*/}"

# parse args
for arg in "$@"
do
    case "$arg" in
        "-v" | "--verbose")
            VERBOSE=1
            ;;
        "-d" | "--debug")
            export DISCORD_DEV_LOG_DEBUG=1
            ;;
        *)
            command="${command} ${arg}"
            ;; 
    esac
    shift
done
$( echo "${command}" )
