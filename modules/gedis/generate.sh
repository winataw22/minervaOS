#!env sh
DIR="$1"
PKG="$2"

function help(){
	echo "Usage: generate.s <schema-dir> <pkg-name>"
	exit 1
}

if [ -z ${DIR} ] || [ -z ${PKG} ]; then
	help
fi

if [ ! -d ${DIR} ]; then 
	echo "[-] invalid schema directory"
	exit 1
fi

export OUT=types/${PKG}
mkdir -p ${OUT}
for FILE in $(ls ${DIR}/*.toml); do 
	echo "Generating: ${FILE}"
	schemac -dir "${OUT}" -pkg "${PKG}" "${FILE}"
done 

