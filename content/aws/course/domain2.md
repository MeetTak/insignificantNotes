---
title: "Domain2"
folder: "aws/course"
created: "2026-05-22T15:11:35.000Z"
modified: "2026-05-22T20:20:03.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3715"
source: "apple-notes"
---

aws shared responsibility model, it deals with the security of the aws acc

gdpr(general data protection regulation)

aws waf(web application firewall): AWS WAF is a web application firewall to help protect your applications from common exploits that could impact your application availability, compromise your security or consume excessive resources.

aws guardDuty: GuardDuty is a threat detection service to monitor for malicious activity and unauthorized behavior.

aws shield: AWS Shield is a managed DDoS protections service to help safeguard your applications running on AWS.

aws cloudWatch: is used for monitoring and collecting operational data

aws cloudTrail: is used to log events related to AWS resource creation and management.

aws config: is used to take inventory of current configurations and audit resources to ensure they maintain correct configurations.

aws audit manager

Amazon Cognito and specifically Amazon Cognito Identity Pool can provide temporary AWS credentials for this use case. Diving deeper, Amazon Cognito User Pool is a user directory in Amazon Cognito and it doesn't enable access to unauthenticated identities. You have to use an Identity Pool instead.

There are different types of IAM policies in AWS, managed policies and unmanaged. You should know the difference between the two types of policies, who can create each type of policy, and who can edit or modify each type of policy.

AWS creates and manages managed policies whereas customers create and manage regular IAM policies

Network access control lists only manage traffic that is crossing that subnet boundary
