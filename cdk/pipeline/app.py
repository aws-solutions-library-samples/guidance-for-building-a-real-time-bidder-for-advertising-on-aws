"""
Description: Guidance for Building a Real Time Bidder for Advertising on AWS (SO9111). 
Deploys AWS CodeBuild and CodePipeline
"""
import os
import aws_cdk as cdk
from cdk_nag import AwsSolutionsChecks, NagSuppressions

from pipeline.pipeline_stack import BuildStack


app = cdk.App()
# pass stage as input as needed
pipeline_stack = BuildStack(app, 
                               "RTBBuildStack",
                                env=cdk.Environment(
                                    account=os.environ["CDK_DEFAULT_ACCOUNT"],
                                    region=os.environ["CDK_DEFAULT_REGION"]),
                                    description="Guidance for Building a Real Time Bidder for Advertising on AWS (SO9111). Deploys AWS CodeBuild and CodePipeline that in turn deploys the CFN templates with infra and bidder application on EKS"
                                )

nag_suppressions = [
        {
            "id": "AwsSolutions-IAM5",
            "reason": "AWS managed policies are allowed which sometimes uses * in the resources like - AWSGlueServiceRole has aws-glue-* . AWS Managed IAM policies have been allowed to maintain secured access with the ease of operational maintenance - however for more granular control the custom IAM policies can be used instead of AWS managed policies",
        },
        {
            "id": "AwsSolutions-IAM4",
            "reason": "AWS Managed IAM policies have been allowed to maintain secured access with the ease of operational maintenance - however for more granular control the custom IAM policies can be used instead of AWS managed policies",
        },
        {
            "id": "AwsSolutions-S1",
            "reason": "S3 Access Logs are enabled for all data buckets. This stack creates a access log bucket which doesnt have its own access log enabled.",
        },
        {
            'id': 'AwsSolutions-KMS5',
            'reason': 'For sample code key rotation is disabled. Customers are encouraged to enable this in their environment',
        },
    ]

NagSuppressions.add_stack_suppressions(
    pipeline_stack,
    nag_suppressions,
    apply_to_nested_stacks=True
)

# cdk.Aspects.of(app).add(AwsSolutionsChecks())
app.synth()
