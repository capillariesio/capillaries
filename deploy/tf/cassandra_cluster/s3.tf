# Temporary storage for provisioning scripts exceedeng 16k
resource "aws_s3_bucket" "tf_deploy_bucket" {
  bucket = var.capillaries_tf_deploy_temp_bucket_name
}

resource "aws_s3_object" "s3_object_bastion_sh" {
  bucket = aws_s3_bucket.tf_deploy_bucket.id
  key    = "bastion.sh"
  source = "scripts/bastion.sh"
  etag   = filemd5("scripts/bastion.sh")
}

resource "aws_s3_object" "s3_object_daemon_sh" {
  bucket = aws_s3_bucket.tf_deploy_bucket.id
  key    = "daemon.sh"
  source = "scripts/daemon.sh"
  etag   = filemd5("scripts/daemon.sh")
}

resource "aws_s3_object" "s3_object_cassandra_sh" {
  bucket = aws_s3_bucket.tf_deploy_bucket.id
  key    = "cassandra.sh"
  source = "scripts/cassandra.sh"
  etag   = filemd5("scripts/cassandra.sh")
}