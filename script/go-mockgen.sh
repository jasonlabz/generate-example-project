#!/usr/bin/env bash

set -euo pipefail

usage() {
	printf 'Usage: %s -o <output-root>\n' "${0##*/}" >&2
}

output_root=''
while (($# > 0)); do
	case "$1" in
	-o)
		if (($# < 2)); then
			usage
			exit 2
		fi
		output_root="$2"
		shift 2
		;;
	*)
		usage
		exit 2
		;;
	esac
done

if [[ -z "$output_root" ]]; then
	usage
	exit 2
fi

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

if [[ "$output_root" =~ ^[A-Za-z]:[\\/].* ]]; then
	output_root="$(cygpath --unix "$output_root")"
elif [[ "$output_root" != /* ]]; then
	output_root="$project_root/$output_root"
fi
mkdir -p "$output_root"

while IFS= read -r source_relative_path; do
	[[ -z "$source_relative_path" ]] && continue

	source_path="$project_root/$source_relative_path"
	source_directory="$(dirname "$source_relative_path")"
	source_filename="$(basename "$source_relative_path" .go)"

	if [[ "$source_relative_path" == internal/* ]]; then
		internal_prefix=''
		internal_suffix="${source_relative_path#internal/}"
	elif [[ "$source_relative_path" == */internal/* ]]; then
		internal_prefix="${source_relative_path%%/internal/*}"
		internal_suffix="${source_relative_path#*/internal/}"
	else
		destination_directory="$output_root/$source_directory"
		mkdir -p "$destination_directory"
		go run go.uber.org/mock/mockgen@v0.6.0 -source "$source_path" -destination "$destination_directory/mock_${source_filename}.go" -package mocks
		continue
	fi

	internal_directory="$(dirname "$internal_suffix")"
	destination_directory="$project_root"
	if [[ -n "$internal_prefix" ]]; then
		destination_directory="$destination_directory/$internal_prefix"
	fi
	destination_directory="$destination_directory/.mocks/internal"
	if [[ "$internal_directory" != '.' ]]; then
		destination_directory="$destination_directory/$internal_directory"
	fi
	mkdir -p "$destination_directory"
	go run go.uber.org/mock/mockgen@v0.6.0 -source "$source_path" -destination "$destination_directory/mock_${source_filename}.go" -package mocks
done < <(go run ./script/internal/mockscan -root "$project_root" -exclude "$output_root")
