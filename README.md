# EC2 instance private IPs to Route53 SRV record via a security group

Inspired by [github.com/rlister/asg-route53](https://github.com/rlister/asg-route53) which discovers IPs for SRV records using an autoscaling group ID. Alternatively, this progam discovers IPs with a security group.

## Usage

```
$ export AWS_REGION=us-east-1
$ export ROOT_ZONE=example.com
$ export FQDN="_etcd-server._tcp.example.com"
$ export SECURITY_GROUP_NAME=etcd
$ bin/ec2-sg-route53
```
