## Build/Deploy
### Prerequisites
In order to deploy the example code, you will need a working, up-to-date AWS
CLI environment, with a profile set up for the account you want to deploy to.

If you're running on a Mac, you must install GNU gettext using Homebrew
(`brew install gettext`) before running the deploy script. (Note that you do
*not* need to use `brew install --force`.) If you use MacPorts
or some other package manager, you'll have to modify the script. Good luck!

### Deploying
To deploy to AWS, from within the project directory (`examples/simple` or
`examples/gorillamux`), run the following command:

```STACK_NAME=<<stack-name>> ../deploy/deploy.sh```

If all goes well, that will deploy all the configuration and artifacts to AWS.
The script will output the URL of the deployed API.

If it fails, you can look at the CloudFormation stacks in the console to see what
went wrong.

The STACK_NAME environment variable is used as the CloudFormation stack name
and also to name various resources created during the deploy. So if you name
your stack `simple-example` the Lambda function will be named `simple-example-lambda`.

### Testing
To test that your deploy worked, you can use a browser or `curl` to access the
example API you deployed. The API Gateway and Lambda logs in the CloudWatch
console should show the result of your test API calls.
