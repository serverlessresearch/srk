import argparse
import json
import matplotlib
import matplotlib.pyplot as plt
import numpy as np


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Plot Serverless Data.')
    parser.add_argument('--experiment-id', required=True, help='which experiment to print')
    parser.add_argument('--log', default='log.txt', help='log file to plot')
    parser.add_argument('--metric-value', default='concurrency', help='what metric to plot')
    args = parser.parse_args()

    experiment_id = args.experiment_id

    run_intervals = []
    reports = {}
    with open(args.log) as f:
        for line in f.readlines():
            event = json.loads(line)
            if "uuid" in event and event["uuid"].startswith(experiment_id):
                if event["action"] == "end":
                    r = (event["uuid"], event["begin_time"], event["end_time"], {})
                    run_intervals.append(r)
                elif event["action"] == "report":
                    reports[event["uuid"]] = event["data"]

    min_start = min([x[1] for x in run_intervals])
    max_end = max([x[2] for x in run_intervals])

    experiment_len = max_end - min_start

    plot_interval = 0.1

    plot_n = int(experiment_len / plot_interval) + 1

    def update_concurrency(vec, index, uuid, dt):
        vec[index] += 1

    def update_metric_sum(vec, index, uuid, dt):
        if uuid in reports:
            data = reports[uuid]
            if args.metric_value in data:
                vec[index] += data[args.metric_value] / dt
            else:
                print("missing key %s in report for uuid %s" % (args.metric_value, uuid))
        else:
            print("missing report for uuid %s" % uuid)

    if args.metric_value == 'concurrency':
        update_fn = update_concurrency
        metric_label = 'concurrency'
    else:
        update_fn = update_metric_sum
        metric_label = "%s (1/s)" % args.metric_value


    v = [0] * plot_n
    for (uuid, start, end, data) in run_intervals:
        start_int = int((start - min_start) / plot_interval)
        end_int = int((end - min_start) / plot_interval)
        for i in range(start_int, end_int):
            update_fn(v, i, uuid, end - start)
    t = np.arange(0, experiment_len, plot_interval)

    fig, ax = plt.subplots()
    ax.plot(t, v)
    ax.set(xlabel='time (s)', ylabel=metric_label)
    ax.grid()
    fig.savefig('%s-%s.pdf' % (experiment_id, args.metric_value))
    # plt.show()
