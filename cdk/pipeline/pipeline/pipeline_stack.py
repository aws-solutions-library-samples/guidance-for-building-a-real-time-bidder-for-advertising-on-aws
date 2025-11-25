# Description: Guidance for Building a Real Time Bidder for Advertising on AWS (SO9111). Deploys AWS CodeBuild and CodePipeline that in turn deploys the CFN templates with infra and bidder application on EKS
import os
from aws_cdk import (
    Stack,
    aws_codebuild as cb,
    aws_iam as iam
)

from constructs import Construct
class BuildStack(Stack):

    def __init__(self, scope: Construct, construct_id: str, stage: str="dev" , **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        acc = os.getenv("CDK_DEFAULT_ACCOUNT")
        reg = os.getenv("CDK_DEFAULT_REGION")

        # Get environment-specific context
        env_context = self.node.try_get_context(stage)
        shared_context = self.node.try_get_context('shared')
        
        repo_owner = self.get_context_value("REPO_OWNER", shared_context, "aws-solutions-library-samples")
        repo_name = self.get_context_value("REPO_NAME", shared_context, "guidance-for-building-a-real-time-bidder-for-advertising-on-aws")
        root_stack_name = self.get_context_value("ROOT_STACK_NAME", shared_context, None)

        if not root_stack_name:
            raise ValueError("ROOT_STACK_NAME is required in the shared context. You can set it with OS Variable ROOT_STACK_NAME")
        else:
            # fix for issue #59 - Bucket name that prefixes stack name needs to be lowercase
            # and cannot have underscores
            root_stack_name = root_stack_name.lower().replace("_", "-")
        
        stack_variant =  self.get_context_value("STACK_VARIANT", shared_context, "DynamoDBBasic")
        repo_branch = self.get_context_value("REPO_BRANCH", env_context, "main")    
        
        # print out all variables above
        print("repo_owner (OS REPO_OWNER): ", repo_owner)
        print("repo_name ($REPO_NAME): ", repo_name)
        print("root_stack_name ($STACK_NAME): ", root_stack_name)
        print("stack_variant: ", stack_variant)
        print("repo_branch: ", repo_branch)
        print("acc: ", acc)
        print("reg: ", reg)
        
        

        # fix for issue #79 code commit deprecation
        # the solution now points to the opensource github repo by default
        # customers can update their repo configurations through context variables
        # To provide GitHub credentials, please either go to AWS CodeBuild Console to connect or call ImportSourceCredentials to persist your personal access token. Example:
        # aws codebuild import-source-credentials --server-type GITHUB --auth-type PERSONAL_ACCESS_TOKEN --token <token_value>
        cb_source = cb.Source.git_hub(
            owner=repo_owner,
            repo=repo_name,
            webhook=False,
            branch_or_ref=repo_branch
        )
        # Defines the artifact representing the sourcecode

        rtb_pipeline_role = iam.Role(self, id="rtbkit_codebuild_role", role_name="rtbkit_codebuild_role",
            assumed_by=iam.CompositePrincipal(
                iam.ServicePrincipal('codebuild.amazonaws.com'),
                iam.ServicePrincipal('codepipeline.amazonaws.com'),
            ),
            path="/rtbkit/"
        )
        # Fix for issue #61
        rtb_pipeline_role = self.add_managed_policies(rtb_pipeline_role)
        
        build_spec = self.get_context_value("BUILD_SPEC", shared_context, "buildspec.yml")

        cb_project = cb.Project(self, "rtb-build-project",
            environment={
                "build_image": cb.LinuxBuildImage.AMAZON_LINUX_2_ARM_3,
                "privileged": True,
            },
            environment_variables={
                "AWS_ACCOUNT_ID": cb.BuildEnvironmentVariable(value=acc),
                "RTBKIT_ROOT_STACK_NAME": (cb.BuildEnvironmentVariable(value=root_stack_name)),
                "RTBKIT_VARIANT": cb.BuildEnvironmentVariable(value=stack_variant),
            },
            build_spec=cb.BuildSpec.from_source_filename(build_spec),
            source=cb_source,
            role=rtb_pipeline_role,
            project_name="rtb-build-project"
        )
        
        # https://stackoverflow.com/questions/63659802/cannot-assume-role-by-code-pipeline-on-code-pipeline-action-aws-cdk
        cfn_build = cb_project.node.default_child
        cfn_build.add_override("Properties.Environment.Type", "ARM_CONTAINER")
    
    # fix for issue #61
    def add_managed_policies(self, iamrole: iam.Role) -> iam.Role:
        """
        loops through the list of role arns and add it to the input role object and returns the same back
        """
        managed_policy_arns={"arn:aws:iam::aws:policy/AdministratorAccess",
                    "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
                    "arn:aws:iam::aws:policy/AmazonS3FullAccess",
                    "arn:aws:iam::aws:policy/AmazonKinesisFullAccess",
                    "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
                    "arn:aws:iam::aws:policy/AmazonVPCFullAccess",
                    "arn:aws:iam::aws:policy/AWSCodeBuildAdminAccess",
                    "arn:aws:iam::aws:policy/AWSCloudFormationFullAccess"}
        
        for i,arn in enumerate(managed_policy_arns):
            mananged_policy = iam.ManagedPolicy.from_managed_policy_arn(
                scope=self,
                id=f"rtbkit_admin_policy_{i}",
                managed_policy_arn=arn
            )
            iamrole.add_managed_policy(mananged_policy)

        return iamrole
    
    def get_context_value(self, key: str, context: any, default_value: any) -> any:
        """
        Gets a value from environment variables first, then CDK context, or returns default value.
        
        Args:
            key: The key to look up
            default_value: The default value to return if key is not found
        
        Returns:
            The value from environment, context, or default value
        """

        env_value = os.environ.get(key)
        if env_value is not None:
            return env_value
            
        # Then check CDK context if context is not none
        if context is not None and key in context:
            return context[key]
            
        # Return default if neither found
        return default_value
    