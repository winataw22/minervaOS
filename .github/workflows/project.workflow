workflow "✨Add new issues to projects" {
  resolves = ["alex-page/add-new-issue-project"]
  on = "issues"
}

action "alex-page/add-new-issue-project" {
  uses = "alex-page/add-new-issue-project@master"
  args = [ "zero-os_2.0.0 (active)", "To do"]
  secrets = ["GITHUB_TOKEN"]
}

workflow "✨Add new pull requests to projects" {
  resolves = ["alex-page/add-new-pulls-project"]
  on = "pull_request"
}

action "alex-page/add-new-pulls-project" {
  uses = "alex-page/add-new-pulls-project@master"
  args = [ "zero-os_2.0.0 (active)", "In Progress"]
  secrets = ["GITHUB_TOKEN"]
}
