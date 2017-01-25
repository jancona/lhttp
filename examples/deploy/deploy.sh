#!/bin/bash
# This script uses the envsubst command which is not installed by default on OS/X.
# If you use Homebrew and install GNU gettext using 'brew install gettext' this script will
# use that envsubst. (Note that you do not have to use 'brew install --force'.) If
# you use MacPorts or some other package manager, you'll have to modify the alias
# below to point to the commend on your system.
#set -x
case $OSTYPE in
# Linux*)
#   ;;
darwin*)
  shopt -s expand_aliases
  alias envsubst=`brew ls gettext | fgrep bin/envsubst`
  ;;
esac
# Build the program and deploy zip
GOOS=linux go build -o main

zip -r lambda.zip main index.js
deploy_version=`date -u +"%Y%m%dT%H%M%SZ"`
s3_bucket_name="${STACK_NAME}-deploy"
if aws s3 ls "s3://${s3_bucket_name}" 2>&1 | grep -q 'NoSuchBucket'
then
  aws s3 mb "s3://${s3_bucket_name}"
fi
echo "Subsituting deploy-specific values into Swagger document..."
export AWS_ACCOUNT_ID=`aws sts get-caller-identity --output text --query 'Account'`
envsubst '$AWS_ACCOUNT_ID' <../deploy/api-swagger.yaml >/tmp/api-swagger.yaml
echo "Copying deploy artifacts to S3..."
aws s3 cp ../deploy/cf-template.yaml "s3://${s3_bucket_name}"
aws s3 cp /tmp/api-swagger.yaml "s3://${s3_bucket_name}"
aws s3 cp lambda.zip "s3://${s3_bucket_name}/lambda-${deploy_version}.zip"
command=update-stack complete_command=stack-update-complete
aws cloudformation describe-stack-resources --stack-name $STACK_NAME || command=create-stack complete_command=stack-create-complete
echo "Invoking CloudFormation ${command} on $STACK_NAME..."
aws cloudformation $command --stack-name $STACK_NAME --template-url "https://s3.amazonaws.com/${s3_bucket_name}/cf-template.yaml" --parameters "ParameterKey=DeployVersion,ParameterValue=${deploy_version}" --capabilities CAPABILITY_IAM
ret=$?
# Wait for the stack to complete
if [ "$ret" = "0" ]; then
  echo "Wait for ${command} on $STACK_NAME to complete..."
  aws cloudformation wait $complete_command --stack-name $STACK_NAME
  ret=$?
  if [ "$ret" = "0" ]; then
    echo -n "API URL: "
    aws cloudformation describe-stacks --stack-name $STACK_NAME  | sed -n 's/.*OutputValue": "\([^"]*\).*/\1/p'
  fi
elif [ "$ret" = "255" ]; then
  # This probably means that the nothing changed, therefore
  # there's nothing to do. So we just ignore it.
  echo "Stack update failed, see message above."
  ret=0
else
  echo "Stack update failed, see message above."
fi
exit $ret
