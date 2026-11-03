---
title: "module3"
folder: "aws"
created: "2025-10-29T07:57:05.000Z"
modified: "2025-11-01T10:23:21.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p2858"
source: "apple-notes"
---

selecting a region

- data governance, legal requirements
- proximity to customers(latency)
- services available within the region
- costs(vary by region)

each region has multiple availability zones
each availability zone is a fully isolated partition of the aws infrastructure

aws uses custom network equipment sourced from multiple original device manufacturers(ODMs). ODMs design and manufacture products based on specifications from a second company. The second company then rebrands the products for sale.

aws provides a global network of points of presence locations
consists of edge locations and a much smaller number of regional edge caches

amazon cloudfront is a content delivery network(CDN) used to distribute content to end users to reduce latency. amazon route 53 is a dns service. requests going to either one of these services will be routed to the nearest edge location automatically in order to lower latency.
aws point of presence are located in major cities of the world, and is used by amazon cloudfront, amazon route 53, aws shield, and aws web application firewall(aws waf) services.
regional edge caches are used by default with amazon cloudfront.

aws global infrastructure features:

- elastic and scalable
- fault tolerant
- high availability with minimal down time

aws global infrastructure can be broken down into 3 elements:

- regions
- availability zones
- points of presence

amazon storage services
amazon s3, amazon ebs, amazon efs, and amazon simple storage service glacier

aws compute services
ec2, ec2 auto scaling, ecs, ec2 container registry, elastic beanstalk, lambda, eks, and fargate

aws database services
relational database service, aurora, redshift, and dynamoDB

aws networking and content delivery services
vpc, elastic cloud balancing, cloudfront, transit gateway, route 53, direct connect, and vpn

aws security, identity, and compliance services
identity and access management, organizations, cognito, artifact, key management service, and shield

aws cost management service category
cost and usage report, budgets, and cost explorer

aws management and governance services
management console, config, cloudwatch, scaling, cli, trusted advisor, well architected tool, and cloudtrail

run this command to retrieve the acc no and ur user id
aws sts get-caller-identity

run this command to get the info abt any ec2 instances that are running
aws ec2 describe-instances

edge locations do not need to be located in the same general area as Regions
