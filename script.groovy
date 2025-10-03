//TODO: =========== Initialize functions ===========
def getParameters() {
    return [
        choice(name: 'ENVIRONMENT', choices: ['STAG', 'PROD'], description: 'Select environment for deployment'),
        booleanParam(name: 'RUN_TESTS', defaultValue: false, description: 'Run tests after build')
    ]
}

//TODO: =========== Build functions ===========
def checkoutSSH(url, branchRegex) {
    checkout([
        $class: 'GitSCM',
        branches: [[name: branchRegex]],
        doGenerateSubmoduleConfigurations: false,
        extensions: [],
        userRemoteConfigs: [[
            url: url,
            credentialsId: 'github-ssh'
        ]]
    ])
}

def buildDockerfile(appName, sha) {
    def tag = (sha == null || sha.trim() == '') ? 'latest' : sha
    if (!sha?.trim()) {
        echo "⚠️ FATAL: Commit SHA is empty, tagging as 'latest'"
    }

    def imageName = "ghcr.io/sep490-project/core-backend"
    def imagePrefix = "${imageName}:${tag}"

    // Remove any old images that start with the same short SHA
    sh """
        old_images=\$(docker images --format '{{.Repository}}:{{.Tag}}' | grep '^${imageName}:${tag}' || true)
        if [ -n "\$old_images" ]; then
            echo "🗑 Removing old images with prefix ${imagePrefix}"
            docker rmi -f \$old_images || true
        fi
    """

    sh """
        docker build \\
            --build-arg APP_NAME=${appName} \\
            --label "org.opencontainers.image.source=https://github.com/SEP490-Project/core-backend" \\
            -t ${imageName}:${tag} .
    """
}


def runTests() {
    sh 'go test ./... -v'
}

def archiveArtifacts(appName, sha, branchName) {
    if (sha == null || sha.trim() == '') {
        error "FATAL: Commit SHA is required for archiving artifacts."
        return
    }
    
    def registry = "ghcr.io"
    def imageName = "ghcr.io/sep490-project/core-backend"

    def sanitizedBranchName = branchName.replaceAll('/', '-')

    def imageWithShaTag = "${imageName}:${sha}"
    def imageWithBranchTag = "${imageName}:${sanitizedBranchName}"
    def imageWithLatestTag = "${imageName}:latest"

    withCredentials([usernamePassword(credentialsId: 'ghcr-access',
                                      usernameVariable: 'GH_USER',
                                      passwordVariable: 'GH_PAT')]) {
        sh """
          # Exit immediately if a command exits with a non-zero status.
          set -e

          echo "Logging in to Docker registry at ${registry}..."
          echo "\$GH_PAT" | docker login ${registry} -u "\$GH_USER" --password-stdin

          echo "Tagging image ${imageWithShaTag} with additional tags..."
          docker tag ${imageWithShaTag} ${imageWithBranchTag}
          docker tag ${imageWithShaTag} ${imageWithLatestTag}

          echo "Pushing tags to the registry..."
          docker push ${imageWithShaTag}
          docker push ${imageWithBranchTag}
          docker push ${imageWithLatestTag}
        """
    }
}

//TODO: =========== Notification functions ===========

/**
 * Sends a formatted notification to a Discord channel via webhook.
 * @param status 'SUCCESS' or 'FAILURE'.
 * @param color The decimal color code for the embed's side border.
 */
def sendDiscordNotification(String status, int color) {
    // Determine the build status message and icon
    def statusMessage = (status == 'SUCCESS') ? "Build Success" : "Build Failed"
    def statusIcon = (status == 'SUCCESS') ? "✅" : "❌"

    // Construct the JSON payload for the Discord embed
    // Using a Groovy multiline string is clean and easy to read.
    def jsonPayload = """
    {
      "username": "Jenkins CI",
      "avatar_url": "https://www.jenkins.io/images/logos/jenkins/jenkins.png",
      "embeds": [
        {
          "title": "${statusIcon} ${statusMessage}: ${env.JOB_NAME} #${env.BUILD_NUMBER}",
          "url": "${env.BUILD_URL}",
          "color": ${color},
          "description": "The latest build has completed. See details below.",
          "fields": [
            {
              "name": "Branch",
              "value": "${env.BRANCH_NAME}",
              "inline": true
            },
            {
              "name": "Environment",
              "value": "${params.ENVIRONMENT}",
              "inline": true
            },
            {
              "name": "Commit SHA",
              "value": "`${GIT_COMMIT.take(7)}`",
              "inline": false
            },
            {
              "name": "Build Logs",
              "value": "[Click here to view the console output](${env.BUILD_URL}console)",
              "inline": false
            }
          ],
          "footer": {
            "text": "Job: ${env.JOB_NAME}"
          },
          "timestamp": "${new Date().format("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'", TimeZone.getTimeZone('UTC'))}"
        }
      ]
    }
    """

    // Use withCredentials to securely access the webhook URL
    withCredentials([string(credentialsId: 'discord-webhook-url', variable: 'DISCORD_WEBHOOK_URL')]) {
        sh """
            # Writing the JSON to a temporary file is safer than passing it directly
            # as a command-line argument, as it avoids shell escaping issues.
            echo '${jsonPayload}' > discord_payload.json

            curl -X POST -H "Content-Type: application/json" \\
                 --data '@discord_payload.json' \\
                 "${DISCORD_WEBHOOK_URL}"
        """
    }
}

def sendSuccessNotification() {
    // Green color in decimal
    sendDiscordNotification('SUCCESS', 3066993)
}

def sendFailureNotification() {
    // Red color in decimal
    sendDiscordNotification('FAILURE', 15158332)
}

//TODO: =========== Utility functions ===========
def deployToEnvironment(environment, appName, sha) {
    switch(environment) {
        case 'STAG':
            echo "Deploying ${appName} to STAGING environment (RUN LOCALLY)"
            sh "docker run -d -p 8080:8080 --name ${appName}_${sha}_STAG ${appName}:${sha}"
            break
        case 'PROD':
            echo "Deploying ${appName} to PRODUCTION environment (NOT SUPPORTED YET)"
            break
        default:
            error "Unknown environment: ${environment}"
    }
}

return this

