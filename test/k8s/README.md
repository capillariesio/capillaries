This is a POC demonstrating Capillaries running in a Kubernetes cluster.

 It assumes that:
 - S3 docker repositories are set up
 - S3 bucket that hold lookup_quicktest_s3 Capillaries script and input data files are provisioned
 - Capillaries docker images were built and uploaded using binaries_build.sh, images_build.sh, images_upload.sh commands
 - Minikube Kuberetes cluster is running

Just run scripts in order.

Tests scripts use data and config files stored in S3. Make sure you have the test bucket and IAM user credentials set up as described in [s3 data access](./s3.md).