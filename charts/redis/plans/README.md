# All six workloads have a data set which is similar. Workloads D and E insert records during the test run.
# Thus, to keep the database size consistent, we apply the following sequence:
#
# 1) Load the database, using workload A's parameter file (workloads/workloada) and the "-load" switch to the client.
# 2) Run workload A (using workloads/workloada and "-t") for a variety of throughputs.
# 3) Run workload B (using workloads/workloadb and "-t") for a variety of throughputs.
# 4) Run workload C (using workloads/workloadc and "-t") for a variety of throughputs.
# 5) Run workload F (using workloads/workloadf and "-t") for a variety of throughputs.
# 6) Run workload D (using workloads/workloadd and "-t") for a variety of throughputs.
#    This workload inserts records, increasing the size of the database.
# 7) Delete the data in the database.  Otherwise, the remaining data of the cluster might affect the results
#    of the following workload.
# 8) Reload the database, using workload E's parameter file (workloads/workloade) and the "-load switch to the client.
# 9) Run workload E (using workloads/workloadd and "-t") for a variety of throughputs.
# This workload inserts records, increasing the size of the database.
#
# For the deletion, instead of destroying the cluster, we destroy and recreate the cluster.