@Library('jenkins-library') _

def label = UUID.randomUUID().toString()
def podTemplateYaml=kubernetesTemplate.kubernetesTemplate('172.16.1.99/tostmp/tos-k8s-base')

timestamps {
  properties([buildDiscarder(
          logRotator(artifactDaysToKeepStr: '', artifactNumToKeepStr: '', daysToKeepStr: '60', numToKeepStr: '100')),
              gitLabConnection('gitlab-172.16.1.41'),
              parameters([string(defaultValue: 'tos-3.0', description: '', name: 'RELEASE_TAG')]),
              pipelineTriggers([])
  ])
  updateGitlabCommitStatus(name: 'ci-build', state: 'pending')
  podTemplate(label: label, yaml: podTemplateYaml) {
    node(label) { container('builder') {
      currentBuild.result = "SUCCESS"

      waitDocker {}

      stage('scm checkout') {
        checkout(scm)
      }
      updateGitlabCommitStatus(name: 'ci-build', state: 'running')

      withEnv([
              'DOCKER_HOST=unix:///var/run/docker.sock',
              'DOCKER_REPO=172.16.1.99',
      ]) {

        try {
          withCredentials([
             usernamePassword(
                     credentialsId: 'tosharbor',
                     passwordVariable: 'DOCKER_PASSWD',
                     usernameVariable: 'DOCKER_USER')
          ]) {

            stage('release build') {
                container('builder') {
                    sh """#!/bin/bash -ex
                      docker login -u \$DOCKER_USER -p \$DOCKER_PASSWD \$DOCKER_REPO
                      make build-antrea
                    """
                }
            }

            stage('push docker and charts') {
                container('builder') {
                    sh """#!/bin/bash -ex
                      docker login -u \$DOCKER_USER -p \$DOCKER_PASSWD \$DOCKER_REPO
                      make docker-push
                      make render
                      make charts-push
                    """
                }
            }
          }
          updateGitlabCommitStatus(name: 'ci-build', state: 'success')
        } catch (e) {
          currentBuild.result = "FAILED"
          updateGitlabCommitStatus(name: 'ci-build', state: 'failed')
          echo 'Err: Incremental Build failed with Error: ' + e.toString()
          throw e
        } finally {
          sendMail2 {
              emailRecipients = "tosdev@transwarp.io"
              attachLog = false
          }
        }
      }
    }}
  }
}

