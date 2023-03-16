#!/bin/bash

CATALOG=${CATALOG:-registry.redhat.io/redhat/redhat-operator-index:v4.12}
CHANNEL=${CHANNEL:-stable}
PACKAGE=${PACKAGE:-sriov-network-operator}

get_bundles(){
	local -n array=$1
	if [[ -z ${catalog_fn} ]]; then
		catalog_fn=$(mktemp)
		echo "Catalog file name: $catalog_fn"
		echo "Type \"catalog_fn=$catalog_fn ${BASH_SOURCE[0]}\" to make it faster next time"
		echo "Rendering the catalog, it will take a minute"
		res=$(opm render $CATALOG > $catalog_fn)
	fi
	array=( $(cat $catalog_fn | jq -r --arg PACKAGE $PACKAGE --arg CHANNEL $CHANNEL  '. | 
		select(.schema=="olm.channel") | select(.package | contains($PACKAGE)) |
		select(.name==$CHANNEL) | .entries[].name') )	
}

select_bundle(){
	local -n array=$1
	local -n result=$2
	echo "Select a bundle: "
	for i in "${!array[@]}"; do
		printf "%s. %s\n" "$i" "${array[$i]}"
	done
	read option
	result=${array[$option]}
}

get_bundle_pull(){
	local -n bundle_name=$1
	local -n bundle_pull=$2
	bundle_pull=$(cat $catalog_fn | jq -r --arg bundle_name $bundle_name --arg PACKAGE $PACKAGE '. | 
		select(.schema=="olm.bundle") | select(.package | contains($PACKAGE)) |
		select(.name==$bundle_name) | .image')

}

download_bundle(){
	local -n p=$1
	local bundle_dir=$(mktemp -d)
	id=$(podman pull -q $p)
	mount=$(podman unshare podman image mount $id)
	podman unshare cp -rf $mount/manifests $mount/metadata $bundle_dir/
	podman unshare podman image unmount $id
	echo $bundle_dir
}

main(){
	local bundles
	get_bundles bundles
	local bundle
	select_bundle bundles bundle
	local pull
	get_bundle_pull bundle pull
	printf "Pull spec:\n"$pull"\n"
	echo "pulling the bundle, it will take a minute"
	bd=$(download_bundle pull |tail -1)
	echo $bd
}

if [[ "${BASH_SOURCE[0]}" = "${0}" ]]; then
	main "${@}"
	exit $?
fi