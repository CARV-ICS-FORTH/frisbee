    # Step 0: bootstrap.
    # Cockroachdb requires knowing the names of the clustered nodes, before they are started !.
    # Run cockroach command to any of the created node.
    # Step 1: Load a new dataset, using the parameters of workload A.
    # We use no throttling to maximize this step and complete it soon.
    # Step 2: Run workload A
    # Step 3: Run workload B
    # Step 4: Run workload C
    # Step 5: Run workload F
    # Step 6: Run workload D.
    # Step 7,8: Reload the data with parameters of workload E.
    # We use the dropdata field to remove all data before test.
    # Step 9:Run workload E



# dependencies:
#  - name: observability
#    version: 0.1.1
#    repository: https://carv-ics-forth.github.io/frisbee/charts
#  - name: sysmon
#    version: 0.1.1
#    repository: https://carv-ics-forth.github.io/frisbee/charts
#  - name: ycsbmon
#    version: 0.1.1
#    repository: https://carv-ics-forth.github.io/frisbee/charts
