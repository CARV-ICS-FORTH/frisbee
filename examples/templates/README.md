# Templates

Kubernetes users often find themselves copying YAML specifications and editing them to their needs. This approach makes
it difficult to go back to the source material and incorporate any improvements made to it. The template allows users to
create a library of frequently-used specifications and reuse them throughout the definition of the experiment.

Templates define minimally constraining skeletons, leaving a bunch of strategically predefined blanks where dynamic data
will be injected to create multiple variants of the specification using different sets of parameters.

In contrast to other templating engines that are evaluated at deployment time (e.g., Helm), \frisbee{} templates are
evaluated in runtime to be usable by calling objects, such as Workflows. 

