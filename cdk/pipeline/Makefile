ifeq ("$(wildcard .env)","")
	INVALID_SETTINGS=$(error No .env file is found. Copy the envtemplate to .env in the root and make sure the settings are valid.)
endif

.ONESHELL:

cdk@setup:
	cd cdk/pipeline
	python3 -m venv .venv
	source .venv/bin/activate
	pip3 install -r requirements.txt
	cdk bootstrap

cdk@deploy:
	cd cdk/pipeline
	source .venv/bin/activate
	cdk deploy --require-approval never

cdk@list:
	cd cdk/pipeline
	source .venv/bin/activate
	cdk list 
