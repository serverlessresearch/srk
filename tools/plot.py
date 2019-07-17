import argparse
import json
import matplotlib
import matplotlib.pyplot as plt
import numpy as np

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Plot Serverless Data.')
    parser.add_argument('--experiment-id', required=True, help='which experiment to print')
    parser.add_argument('--log', default='log.txt', help='log file to plot')
    args = parser.parse_args()

    experiment_id = args.experiment_id

    run_intervals = []
    with open(args.log) as f:
        for line in f.readlines():
            event = json.loads(line)
            if "uuid" in event and event["uuid"].startswith(experiment_id):
                if event["action"] == "end":
                    run_intervals.append((event["begin_time"], event["end_time"]))
    min_start = min([x[0] for x in run_intervals])
    max_end = max([x[1] for x in run_intervals])

    experiment_len = max_end - min_start

    plot_interval = 0.1

    plot_n = int(experiment_len / plot_interval) + 1

    concurrency = [0] * plot_n
    for (start, end) in run_intervals:
        start_int = int((start - min_start) / plot_interval)
        end_int = int((end - min_start) / plot_interval)
        for i in range(start_int, end_int):
            concurrency[i] += 1
    t = np.arange(0, experiment_len, plot_interval)
    # print(min_start, max_end, experiment_len, plot_n)
    # print(len(concurrency))
    # print(len(t))
    fig, ax = plt.subplots()
    ax.plot(t, concurrency)
    ax.set(xlabel='time (s)', ylabel='concurrency')
    ax.grid()
    fig.savefig('%s-concurrency.pdf' % experiment_id)
    # plt.show()



