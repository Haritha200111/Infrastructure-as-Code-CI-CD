provider "aws" {
  region = "us-east-2"
}

resource "aws_instance" "my_ec2" {
  ami           = "ami-08ca1d1e465fbfe0c"
  instance_type = "t2.micro"

  tags = {
    Name = "TerraformInstance"
  }
}
