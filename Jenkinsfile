pipeline {
    agent any

    options {
        skipDefaultCheckout(true)
        buildDiscarder(logRotator(numToKeepStr: '5'))
        timeout(time: 15, unit: 'MINUTES')
        timestamps()
    }

    environment {
        HOMECTL = "/Users/youngchaeseo/bin/homectl"
        PATH    = "/Users/youngchaeseo/bin:/Users/youngchaeseo/.go/current/bin:/usr/local/go/bin:/Users/youngchaeseo/go/bin:/Users/youngchaeseo/.local/bin:/usr/local/bin:${env.PATH}"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Quality Gates') {
            steps {
                sh '''
                    set -eu
                    task vet
                    task test
                '''
            }
        }

        stage('Deploy (homectl)') {
            steps {
                sh '''
                    set -eu
                    "${HOMECTL}" deploy public-api-mcp \
                        --source "${WORKSPACE}" \
                        --id "release-${BUILD_NUMBER}" \
                        --keep 5 \
                        --yes
                '''
            }
        }
    }

    post {
        failure {
            echo "❌ 배포 실패: homectl logs public-api-mcp -e 로 확인하세요."
        }
        success {
            echo "🎉 배포 완료: homectl status public-api-mcp"
        }
    }
}
