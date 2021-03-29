#!/bin/bash
set -e

# by default, we compile with (number of cpu threads + 1)
# you can changes this to reduce computer load
JOBS=$(($(grep -c 'bogomips' /proc/cpuinfo) + 1))
MAKEOPTS="-j ${JOBS}"

ROOTPKG="${PWD}/packages"

USE_MIRROR=0
DO_MRPROPER=0

arguments() {
    OPTS=$(getopt -o p:rh --long package:,mrproper,help -n 'parse-options' -- "$@")
    if [ $? != 0 ]; then
        echo "Failed parsing options" >&2
        exit 1
    fi

    if [ "$OPTS" != " --" ]; then
        while true; do
            case "$1" in
                -p | --package)
                    shift;
                    package=$1
                    shift ;;

                -r | --mrproper)
                    DO_MRPROPER=1
                    shift ;;

                -h | --help)
                    echo "Usage:"
                    echo " -p --package <pkg>   process specific package"
                    echo " -r --mrproper        remove all files not provided by default"
                    echo " -h --help            display this help message"
                    exit 1
                shift ;;

                -- ) shift; break ;;
                * ) break ;;
            esac
        done
    fi
}

pushd() {
    command pushd "$@" > /dev/null
}

popd() {
    command popd "$@" > /dev/null
}

setup() {
    green="\033[32;1m"
    orange="\033[33;1m"
    blue="\033[34;1m"
    cyan="\033[36;1m"
    white="\033[37;1m"
    clean="\033[0m"
}

success() {
    echo -e "${green}$1${clean}"
}

info() {
    echo -e "${blue}$1${clean}"
}

warning() {
    echo -e "${orange}$1${clean}"
}

event() {
    echo -e "[+] ${blue}$1: ${clean}$2 ${orange}$3${clean}"
}

#
# Check the md5 hash from a file ($1) and compare with $2
#
checksum() {
    checksum=$(md5sum "$1" | awk '{ print $1 }')

    if [ "${checksum}" == "$2" ]; then
        # echo "[+] checksum match"
        return 0
    else
        echo "[-] checksum mismatch"
        return 1
    fi
}

#
# Sanity check
#
prepare() {
    echo "[+] initializing build tool"
}

#
# Download a file and check the hash
# First argument needs to be the url, second is the md5 hash
#
download_file() {
    # set extra filename output or default output
    if [ ! -z $3 ]; then
        output=$3
    else
        output=$(basename "$1")
    fi

    # set default url
    fileurl=$1

    # if we use a mirror we rewrite the url
    if [ $USE_MIRROR == 1 ]; then
        # if we use a custom filename, that
        # filename will be used on the mirror site
        if [ ! -z $3 ]; then
            fileurl=$3
        fi

        fileurl="$MIRRORSRC/$(basename $fileurl)"
    fi

    event "downloading" "${output}"

    if [ -f "${output}" ]; then
        # Check for md5 before downloading the file
        checksum ${output} $2 && return
    fi

    # Download the file
    if [ "${INTERACTIVE}" == "false" ]; then
        curl -L -k -o "${output}" $fileurl
    else
        curl -L -k --progress-bar -C - -o "${output}" $fileurl
    fi

    # Checksum the downloaded file
    checksum ${output} $2 || false
}

download_git() {
    repository="$1"
    branch="$2"
    tag="$3"

    target=$(basename "${repository}")
    logfile="${target}-git.log"

    echo "Loading ${repository}" > "${logfile}"

    [[ -d "${target}" ]] && rm -rf ${target}

    event "cloning" "${repository}" "[${branch}]"
    git clone -b "${branch}" "${repository}"
}

#
# Dynamic libraries management
#
ensure_libs() {
    echo "[+] verifing libraries dependancies"
    pushd "${ROOTDIR}"

    mkdir -p usr/lib

    if [ ! -e lib64 ]; then ln -s usr/lib lib64; fi
    if [ ! -e lib ]; then ln -s lib64 lib; fi

    export LD_LIBRARY_PATH=${ROOTDIR}/lib:${ROOTDIR}/usr/lib

    # Copiyng ld-dependancy
    ld=$(ldd /bin/bash | grep ld-linux | awk '{ print $1 }')
    cp -aL $ld lib/

    for file in $(find -type f -executable); do
        # Looking for dynamic libraries shared
        libs=$(ldd $file 2>&1 | grep '=>' | grep -v '=>  (' | awk '{ print $3 }' || true)

        # Checking each libraries
        for lib in $libs; do
            libname=$(basename $lib)

            # Library found and not the already installed one
            if [ -e lib/$libname ] || [ "$lib" == "${PWD}/usr/lib/$libname" ]; then
                continue
            fi

            # Grabbing library from host
            cp -avL $lib lib/
        done
    done

    popd
}

exclude_libs() {
    echo "[+] excluding host critical libs"
    rm -rf ${ROOTDIR}/usr/lib/ld-linux*
    rm -rf ${ROOTDIR}/usr/lib/libc.*
    rm -rf ${ROOTDIR}/usr/lib/libdl.*
    rm -rf ${ROOTDIR}/usr/lib/libpthread.*
}

github_name() {
    # force github print
    echo "::set-output name=name::${1}"
    echo "[+] github exported name: ${1}"
}

setpackage() {
    echo "[+] setting up package environment"
    pkg=$1

    # where build script is located
    PKGDIR="${ROOTPKG}/${pkg}"

    # files shipped with buildscript
    FILESDIR="${PKGDIR}/files"

    # temporary directory where to build stuff
    WORKDIR="${PWD}/workdir/${pkg}"

    # where final root files will be places
    ROOTDIR="${PWD}/releases/${pkg}"

    # where downloaded files will be stored
    DISTDIR="${PWD}/distfiles/${pkg}"

    mkdir -p "${WORKDIR}"
    mkdir -p "${ROOTDIR}"
    mkdir -p "${DISTDIR}"

    rm -rf "${WORKDIR}/*"
    rm -rf "${ROOTDIR}/*"
    rm -rf "${DISTDIR}/*"

    echo "[+] package script: $PKGDIR"
    echo "[+] package files: $FILESDIR"
    echo "[+] package workdir: $WORKDIR"
    echo "[+] package rootdir: $ROOTDIR"
    echo "[+] package distdir: $DISTDIR"

    # sourcing package script
    . ${PKGDIR}/${pkg}.sh
}

mrproper() {
    echo "[+] removing downloaded files"
    rm -rf ${PWD}/distfiles/*

    echo "[+] removing releases"
    rm -rf ${PWD}/releases/*

    echo "[+] removing staging files"
    rm -rf ${PWD}/workdir/*
}

getsize() {
    du -shc --apparent-size $1 | tail -1 | awk '{ print $1 }'
}

main() {
    setup

    info "===================================="
    info "=  Zero-OS Extra Binaries Builder  ="
    info "===================================="
    echo ""

    prepare
    arguments $@

    if [ $DO_MRPROPER == 1 ]; then
        mrproper
        exit 0
    fi

    if [ ! -z ${package} ]; then
        if [[ -d "${ROOTPKG}/${package}" ]]; then
            echo "[+] building package: ${package}"

            setpackage ${package}
            build_${package}
            ensure_libs
            exclude_libs

            exit 0

        else
            echo "[-] unknown package: ${package}"
            exit 1
        fi
    fi

    echo "[+] nothing to do"
    exit 1
}

main $@
