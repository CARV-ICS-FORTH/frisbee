# Testplans

A **Test Plan** documents the strategy that will be used to verify a specific test for
a [software](https://en.wikipedia.org/wiki/Software) product.

The plan typically contains a detailed understanding of the eventual [workflow](https://en.wikipedia.org/wiki/Workflow)
that models the deployment scenario for a software product, the test strategy, the resources required to perform
testing, and the Key Performance Indicators required to ensure that a product or system meets its design specifications
and other requirements.

By codifying the test plan in a YAML-based syntax, Frisbee carriers three main benefits to teams:

1. Help people outside the test team such as developers, business managers, customers **understand** the details of
   testing.

2. Test Plan **guides** our thinking. It is like a rule book, which needs to be followed.

3. Important aspects like test estimation, test
   scope,[Test Strategy](https://www.guru99.com/how-to-create-test-strategy-document.html)are **documented** in Test
   Plan, so it can be reviewed by Management Team and re-used for other projects.

A test plan may include a strategy for one or more of the following:

* Baseline: to be performed during the development or approval stages of the product, typically on a small sample of
  units.
* Stress: to be performed during preparation or assembly of the product, in an ongoing manner for purposes of
  performance verification and quality control.

## Baseline

A baseline is **a fixed point of reference that is used for comparison purposes**.

#### YCSB

Cloud Serving Benchmark (YCSB) is **an open-source specification and program suite for evaluating retrieval and
maintenance capabilities of computer programs**. It is often used to compare relative performance of NoSQL database
management systems.

All six workloads have a data set which is similar. Workloads D and E insert records during the test run.

Thus, to keep the database size consistent, we apply the following sequence:

0. Bootstrap the database.

1. Load the database, using workload A's parameter file (workloads/workloada) and the "-load" switch to the client.
2. Run workload A (using workloads/workloada and "-t") for a variety of throughputs.
3. Run workload B (using workloads/workloadb and "-t") for a variety of throughputs.
4. Run workload C (using workloads/workloadc and "-t") for a variety of throughputs.
5. Run workload F (using workloads/workloadf and "-t") for a variety of throughputs.
6. Run workload D (using workloads/workloadd and "-t") for a variety of throughputs. This workload inserts records,
   increasing the size of the database.
7. Delete the data in the database. Otherwise, the remaining data of the cluster might affect the results of the
   following workload. For the deletion, instead of destroying the cluster, we destroy and recreate the cluster.
8. Reload the database, using workload E's parameter file (workloads/workloade) and the "-load switch to the client.
9. Run workload E (using workloads/workloadd and "-t") for a variety of throughputs. This workload inserts records,
   increasing the size of the database.

In general, these steps remain the same for the various databases. The difference is how we bootstrap each database.

#### FIO

To *benchmark* persistent disk performance, use *FIO* instead of other disk *benchmarking* tools such as dd .

Fio spawns a number of threads or processes doing a particular type of I/O action as specified by the user.

## Stress

## Scaleout

## Elasticity

## Chaos
