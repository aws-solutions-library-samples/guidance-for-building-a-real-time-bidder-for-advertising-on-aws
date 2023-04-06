# Description: Guidance for Building a Real Time Bidder for Advertising on AWS (SO9111). Deploys AWS CodeCommit, CodeBuild and CodePipeline

from aws_cdk import (
    core as cdk,
    aws_codecommit as cc,
    aws_codebuild as cb,
    aws_codepipeline as cp,
    aws_codepipeline_actions as cpa,
    aws_iam as iam,
    pipelines as pp
)


# class PipelineStage(cdk.Stage):

#     def __init__(self, scope: cdk.Construct, id: str, **kwargs):
#         super().__init__(scope, id, **kwargs)

#         service = PypipworkshopStack(self, 'WebService')



class PipelineStack(cdk.Stack):

    def __init__(self, scope: cdk.Construct, construct_id: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        stage = self.node.try_get_context("stage")
        
        if not stage:
            stage = "dev"
            
        stage_env_vars = self.node.try_get_context(stage)

        repo = cc.Repository(
            self, 'RTBCodeKit',
            repository_name="RTBCodeKitRepo"
        )

        cb_source = cb.Source.code_commit(repository=repo)

        # Defines the artifact representing the sourcecode
        source_artifact = cp.Artifact()
        # Defines the artifact representing the cloud assembly
        # (cloudformation template + all other assets)
        cloud_assembly_artifact = cp.Artifact()

        # Generates the source artifact from the repo we created in the last step
        source_action=cpa.CodeCommitSourceAction(
            action_name='CodeCommit', # Any Git-based source control
            output=source_artifact, # Indicates where the artifact is stored
            repository=repo # Designates the repo to draw code from
        )

        rtb_pipeline_role = iam.Role(self, id="rtbkit_codebuild_role",
            assumed_by=iam.CompositePrincipal(
                iam.ServicePrincipal('codebuild.amazonaws.com'),
                iam.ServicePrincipal('codepipeline.amazonaws.com'),
            ),
            description=None,
            external_id=None,
            external_ids=None,
            inline_policies=None,
            managed_policies=None,
            max_session_duration=None,
            path="/rtbkit/",
            permissions_boundary=None,
            role_name=None,
        )
        # Fix for issue #61
        # admin_policy = iam.ManagedPolicy.from_managed_policy_arn(
        #     scope=self,
        #     id="rtbkit_admin_policy",
        #     managed_policy_arn={"arn:aws:iam::aws:policy/AdministratorAccess",
        #                         "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
        #                         "arn:aws:iam::aws:policy/AmazonS3FullAccess",
        #                         "arn:aws:iam::aws:policy/AmazonKinesisFullAccess",
        #                         "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess",
        #                         "arn:aws:iam::aws:policy/AmazonVPCFullAccess",
        #                         "arn:aws:iam::aws:policy/AWSCodeBuildAdminAccess",
        #                         "arn:aws:iam::aws:policy/AWSCloudFormationFullAccess"}
        # )

        # rtb_pipeline_role.add_managed_policy(admin_policy)
        rtb_pipeline_role = self.add_managed_policies(rtb_pipeline_role)

        cb_project=cb.Project(self, "RTBPipelineProject",
            environment={
                # "compute-type":
                # "build_image": cb.LinuxBuildImage.from_code_build_image_id("aws/codebuild/standard:5.0"),
                "build_image": cb.LinuxBuildImage.from_code_build_image_id("aws/codebuild/amazonlinux2-aarch64-standard:2.0"),
                # optional certificate to include in the build image
                # "certificate": {
                #     "bucket": s3.Bucket.from_bucket_name(self, "Bucket", "my-bucket"),
                #     "object_key": "path/to/cert.pem"
                # }
                "privileged": True,
            },
            environment_variables={
                "AWS_ACCOUNT_ID": cb.BuildEnvironmentVariable(value=stage_env_vars['AWS_ACCOUNT_ID']),
                # fix for issue #59 - Bucket name needs to be lowercase
                "RTBKIT_ROOT_STACK_NAME": (cb.BuildEnvironmentVariable(value=stage_env_vars['RTBKIT_ROOT_STACK_NAME'].lower().replace("_", "-"))),
                "RTBKIT_VARIANT": cb.BuildEnvironmentVariable(value=stage_env_vars['RTBKIT_VARIANT']),
            },
            source=cb_source,
            role=rtb_pipeline_role,
        )


        pipeline = cp.Pipeline(
            self, 'rtb-pipeline',
            pipeline_name='RTBPipeline',
            stages=[
                cp.StageProps(stage_name="Source", actions=[source_action]),
                cp.StageProps(
                    stage_name="Build",
                    actions=[
                        cpa.CodeBuildAction(
                            action_name="Build",
                            # Configure your project here
                            project=cb_project,
                            input=source_artifact,
                            role=rtb_pipeline_role,
                        )
                    ],
                ),
            ],
            role=rtb_pipeline_role,
        )

        # https://stackoverflow.com/questions/63659802/cannot-assume-role-by-code-pipeline-on-code-pipeline-action-aws-cdk
        cfn_pipeline = pipeline.node.default_child
        cfn_pipeline.add_deletion_override("Properties.Stages.1.Actions.0.RoleArn")
        cfn_pipeline.add_deletion_override("Properties.Stages.2.Actions.0.RoleArn")
        cfn_pipeline.add_deletion_override("Properties.Stages.3.Actions.0.RoleArn")

        cfn_build = cb_project.node.default_child
        cfn_build.add_override("Properties.Environment.Type", "ARM_CONTAINER")
        
        # cfn_codebuild = cb_project.node.default_child
        # cfn_codebuild.add_deletion_override("Properties.ServiceRole")

            # cloud_assembly_artifact=cloud_assembly_artifact,

            # # Builds our source code outlined above into a could assembly artifact
            # synth_action=pp.ShellScriptAction(
            #     action_name="Build",
            #     # build_spec="buildspec.yml"
            #     additional_artifacts=[
            #             source_artifact,
            #     ],
            #     commands=[
            #         "curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash",
            #         "echo \"installed helm\"",
            #         'export DOCKER_CLI_EXPERIMENTAL=enabled',
            #         'echo "Starting build `date` in `pwd`',
            #     #    'chmod +x ./initialize-repo.sh && ./initialize-repo.sh <Account ID> ${REGION} ${STACK_NAME} ${VARIANT} yes yes',
            #         'chmod +x ./initialize-repo.sh && ./initialize-repo.sh 504382435436 us-east-1 rtb-bidder-stack-1006 DynamoDB yes yes',
            #         'echo "Build completed `date`"',
            #     ],
            # )

        # deploy = RTBPipelineStage(self, 'Deploy')
        # pp.add_application_stage(deploy)
    
    # fix for issue #61
    def add_managed_policies(self, iamrole: iam.Role) -> iam.Role:
        """
        loops through the list of role arns and add it to the input role object and retuns the same back
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