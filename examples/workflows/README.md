curl -sSL https://mirrors.chaos-mesh.org/v1.2.2/install.sh | bash -s -- --microk8s

-- Conditional branches

https://githubcom/chaos-mesh/chaos-mesh/blob/master/examples/workflow/custom-taskyaml:


conditionalBranches:
- target: workflow-stress-chaos
  expression: 'exitCode == 0 && stdout == "branch-a"'
- target: workflow-network-chaos
  expression: 'exitCode == 0 && stdout == "branch-b"'
- target: on-failed
  expression: 'exitCode != 0