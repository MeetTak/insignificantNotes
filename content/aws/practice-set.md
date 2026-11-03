---
title: "practice set"
folder: "aws"
created: "2026-04-26T10:48:42.000Z"
modified: "2026-04-28T07:23:25.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3639"
source: "apple-notes"
---

spot instance
ec2 users to upgrade their service at discounted rate

cloudFront
hosting static website from a single amazon s3 bucket, amazon service to achieve lower latency and high transfer speeds, content is cached in edge locations, content that is repeatedly accessed can be served from the edge locations instead of the source S3 bucket

two responsibility according to aws shared responsibility model
configure iam users according to the principle of least privilege
configure an amazon s3 bucket to allow public access

amazon aurora
is rdbs build for the cloud service provided by amazon

aws trusted advisor that checks security groups for rules that allow unrestricted access to a resource. unrestricted access increases opportunities for malicious activity.

company must used site-to-site vpn creates an encrypted network path b/w ur on-premises network and ur aws cloud network

aws mgn is an automated lift and shift solution. this solution can migrate physical servers and any databases or applications that run on them to ec2 instances in aws

amazon polly
text to speech tool

aws direct connect
it links ur internal network to a direct connect location through a standard ethernet fiber optic cable. one end of the cable connects to ur router. the other end of the cable connects to a direct connect router.

create an organization in aws organizations with all features enabled within one account invite all accounts to join the organization

amazon macie is an automated security assessment service that helps improve the security and compliance of applications deployed on aws

benefits of using the aws cloud for companies with customers in many countries around the world
companies can deploy applications in multiple aws regions to reduce latency
amazon cloudfront has multiple edge locations around the world to reduce latency

advantages of deploying an application with amazon ec2 instances in multiple availability zones
preventing a single point of failure
increasing the availability of the application

reliability pillar includes the ability of a workload to perform its intended function correctly and consistently when it is expected to do so. the deployment of amazon rds in multiple availability zones supports the goal of reliability because it reduces single points of failure

amazon simple notification service
quality assurance check notification

aws lambda usage crosses the free tier, is charge based on
by the time it takes for the lambda function to run
by the number of requests made for a given lambda function

aws certificate manager
used to implement the encryption in transit

amazon s3 characteristics
an obj store
a durable storage system

credential components required to gain programmatic access to aws account
an access key id
a secret access key
