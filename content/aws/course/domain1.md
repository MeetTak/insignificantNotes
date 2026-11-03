---
title: "Domain1"
folder: "aws/course"
created: "2026-05-20T07:00:39.000Z"
modified: "2026-05-20T11:41:54.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3711"
source: "apple-notes"
---

vertical scaling
increasing the size of the instance to handle the load i.e. adding more cpu or ram
only have one copy of ur application

horizontal scaling
adding more instances of the same size to handle that load
u have more than one copy of ur application for each instance and can use a load balancer to distribute the load across all of ur instances

elasticity is using automation along with horizontal scaling to match capacity and supply with the demand. aws provide launch config’s and auto scaling to scale out ur systems to match that capacity to that demand, which enables ur env to scale out, adding additional resources as the demand increases.And then when that demand decreases, we can scale back in to a smaller number of servers or even no servers, which optimizes for performance efficiency, operational excellence, as well as cost optimization, which are all pillars of the AWS Well-Architected Framework.

7 migration strategies
Retire is for applications you want to decommission or retire. Retain is for applications that you want to keep in the source environment or applications that are not ready to migrate. Rehost, also known as lift and shift are for applications to migrate without making any changes to the application. Relocate is for a large number of servers that are made up of one or more applications. Repurchase is also known as drop and shop and is for applications with a different version or product and provides more value than the existing infrastructure. Replatform is also known as lift, tinker, and shift and is for applications that need some level of optimization in order to operate efficiently or take advantage of AWS capabilities. Refactor or re-architect is for applications that you want to migrate to AWS and take full advantage of the cloud-native features to improve agility, performance, and scalability.

note:
If you launch an Amazon EC2 instance using a Windows AMI, you can choose one bundled with SQL Server Standard, and you do not need to purchase your own license from Microsoft. You could possibly choose the Amazon RDS database instance that runs SQL Server Standard and purchase the license, but the question asked for cost optimization and Amazon RDS costs more than Amazon EC2 because Amazon RDS is a managed service. Also, Amazon Aurora only supports MySQL and PostgresSQL, not SQL Server. aws s3 glacier is used for backup for storing large amounts of data.

4 parts of TCO(total cost of ownership)

- OpEx(operational expenses): day to day operating cost
- CapEx(capital expenses): which are cost associated with creating the longer-term benefits
- Laber calls: staffing an on-premises env or data center, such as the network operation center technicians
