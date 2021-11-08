# Templates

Kubernetes' users often find themselves copying YAML specifications and editing them to their needs. This approach makes
it difficult to go back to the source material and incorporate any improvements made to it. The template allows users to
create a library of frequently-used specifications and reuse them throughout the definition of the experiment.

Templates are a list of services specifications that are allowed to reference and initialize.

Templates define minimally constraining skeletons, leaving a bunch of strategically predefined blanks where dynamic data
will be injected to create multiple variants of the specification using different sets of parameters.

In contrast to other templating engines that are evaluated at deployment time (e.g., Helm), Frisbee templates are
evaluated in runtime to be usable by calling objects, such as Workflows.

    A template directive is enclosed in {{ and }} blocks.

The template directive {{ .Release.Name }} injects the release name into the template. The values that are passed into a
template can be thought of as namespaced objects, where a dot (.) separates each namespaced element.