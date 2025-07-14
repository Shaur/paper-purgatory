pipeline {
  agent {
      label "kubeagent"
  }
  options {
    skipStagesAfterUnstable()
    skipDefaultCheckout()
  }
  stages {
    stage("Prepare container") {
      stages {
      stage('SCM') {
        steps {
            checkout scm
           }
        }
        stage('Docker build') {
            steps {
                sh 'docker build -t paper-purgatory .'
                sh "docker tag paper-purgatory ${DOCKER_REGISTRY}/paper-purgatory:latest"
                sh "docker push ${DOCKER_REGISTRY}/paper-purgatory:latest"
            }
        }
        stage('Helm deploy') {
            steps {
                withKubeConfig([serverUrl: "${CLUSTER_URL}", namespace: "default"]) {
                    sh 'helm upgrade --install paper-purgatory paper-chart'
                    sh 'kubectl rollout restart deployment paper-purgatory'
                }
            }
        }
      }
    }
  }
}