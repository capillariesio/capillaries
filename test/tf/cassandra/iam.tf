# Role that gives Capillaries daemon instances full access to all S3 buckets

resource "aws_iam_role" "capillaries_assume_role" {
  name = "capillaries_assume_role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

tags = {
   Name = "capillaries_assume_role"
 }
}

# For more granular S3 access, create a new policy
resource "aws_iam_role_policy_attachment" "capillaries_s3_access_policy_attachment" {
  role       = aws_iam_role.capillaries_assume_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
}
