import setuptools


with open("README.md") as fp:
    long_description = fp.read()


setuptools.setup(
    name="pipeline",
    version="0.0.2",

    description="RTB Code kit CDK code build pipeline",
    long_description=long_description,
    long_description_content_type="text/markdown",

    author="AWS AMT TFC",

    package_dir={"": "pipeline"},
    packages=setuptools.find_packages(where="pipeline"),

    install_requires=[
        "aws-cdk-lib==2.149.0",
        "cdk-nag==2.28.195"
    ],

    python_requires=">=3.9",

    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Programming Language :: JavaScript",
        "Programming Language :: Python :: 3 :: Only",
        "Topic :: Software Development :: Code Generators",
        "Topic :: Utilities",
        "Typing :: Typed",
    ],
)
