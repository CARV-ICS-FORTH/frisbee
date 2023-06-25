
## Sequential Tests
This examples demonstrates how to use frisbee as a CI. The experiment consists of multiple tests
that run sequentially (on different pods) and stress the Parallax key/value store.

```shell
kubectl frisbee submit test parallax- examples/ci/sequential-tests.yaml charts/databases/parallax/ charts/system/
```