## Guide for the Frisbee Plan Developers

* Spurious Alert may be risen if the expr evaluation frequency is less than the scheduled interval.
* In this case, Grafana faces an idle period, and raises a NoData Alert.
* The controller ignores such messages.