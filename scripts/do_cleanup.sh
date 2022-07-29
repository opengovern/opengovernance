#!/bin/bash

min_to_keep=${min_to_keep:-5}
delete_days_old=${delete_days_old:-4}
verbose=no
repository=''

while [ $# -gt 0 ]; do

	  if [[ $1 == *"--"* ]]; then
		      param="${1/--/}"
		          declare "$param"="$2"
			    fi

			      shift
		      done

		      [[ -z "$repository" ]] && {
			        echo "Error: Need to define '--repository'"
		        exit 1
		}

	date_boundary=$(date --date="$delete_days_old day ago" +%s)
	deletable_tags=$(doctl registry repository list-tags "$repository" --output json | jq ".[5:] | .[] | select ( .updated_at | fromdateiso8601 < $date_boundary) | .tag " -r | tr '\n' ' ')

	[[ -z "$deletable_tags" ]] && {
		  [[ "$verbose" == "yes" ]] && echo "Nothing to delete"
	  exit 0
  }

[[ "$verbose" == "yes" ]] && {
	  echo "We are going to DELETE images that are more than $delete_days_old days"
  echo "We are going to KEEP at least the newer $min_to_keep images"

    echo "Deleteable tags: $deletable_tags"
}
doctl registry repository delete-tag "$repository" $deletable_tags --force
