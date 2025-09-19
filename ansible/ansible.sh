#!/bin/bash

set -e

if [[ -d .venv ]]
then
	source .venv/bin/activate
else
	python3 -m venv .venv
	source .venv/bin/activate
fi

.venv/bin/pip install -r requirements.txt
.venv/bin/ansible-galaxy install -r "$(dirname "$0")/requirements.yml"

export ANSIBLE_FORCE_COLOR=1
export PYTHONUNBUFFERED=1

.venv/bin/ansible-playbook "$@"
