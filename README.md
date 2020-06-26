# Github Action Template Generator

This tool uses a predefined Github action template to generate Github action pipeline based on the given template. See an example [template for java microservice to be deployed to AWS Kube cluster](templates/java-template.yml). This tool is very useful so as to ensure a consistent way of generating Github Action pipeline for given a group of services


## Why should you use this too?

The alternative to not automate `ci pipeline generation` across all the deployements is to copy and paste. We all know what `c&p` means, it is a manual process and prone to making errors. This tool will perform the following for you.

* There is only one source of the truth for the ci pipeline. Instances of the ci pipeline are gtenerated for a given deployment from this template
* Generates multiple instances of ci confi pipeline in one go from a given config file.
* Automatically creates a Pull request for all committed changes to Github.


## Usage

### Configuration files
This toll relies on two configuration files: 

1. Configuration file that defines all the deployments with their corresponding Git repos - see [config.yml](config.yml). Here is a snippet a code

```
description: Generic template to generate github action configuration files for the repositories defined in the config file
repositories:
  - repo:
      url: https://github.com/cirtak/test_repo
      lang: java
  - repo:
      url: https://github.com/cirtak/test_repo_dummy
      lang: java   
```      

The above configuration file contains list of the services or deployments.


2. Template file - see [java-action-template.yml as example of java microservice to be deployed to AWS kube](templates/java-action-template.yml).


### Running the tool

1. Export the following as environment varialbes

```
export GITHUB_TOKEN=xxx
export GPG_PASSWORD=xxx - only needed if GPG signing is enalbed
export GITHUB_EMAIL=xxx - this is the email address associated with your github account
```

2. Generate GPG private key if GPG signing is enabled

You can generate private key as follow. First extract the `GPG secret key` by running the following command: 

`gpg --list-secret-keys --keyid-format LONG`

The output of the above should be something similar to this:

```
sec   rsa4096/B3FBB1D34AA9E501 2020-06-18 [SC] [expires: 2022-06-18]
      1A4945B7F183D631AB99D982B3FBB1D34AA9E501
uid                 [ultimate] ayache@cirta.dev <ayache@cirta.dev>
```

The secret key should be `B3FBB1D34AA9E501`


Then use the above secret key to generate GPG private key

```
gpg --armor --export-secret-keys ${ID} > gpg-private-key 
```

3. Now you can run the tool with the following command: `go build && ./ci-templater`


### Run the tool in dry mode

The defauilt set up of this tool is that it creates a pull request for each generated template. You can disable this by setting the `DryRun` flag as follow when running the tool

` go build && ./ci-templater -DryRun=true`


### Generating the template for only one given depoloyment

By default the tool will extract the deployment details from `config.yml` then uses the appropriate template for each deployment. To apply the filter for only one depoloyment run the tool witht the `repo` flag

`go build && ./ci-templater -repo=test_repo`


## Limitations

Currently this tools assume the generated ci pipelines is going to be pushed to Github repositoy. Support for other repositories will be added in due course. Enjoy!!



