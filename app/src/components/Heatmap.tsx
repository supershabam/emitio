import * as React from "react";
import HeatmapHistogram from "./HeatmapHistogram";

const Heatmap = props => {
  let histograms = props.heatmap.histograms.map((buckets, index) => {
    return (
      <svg
        viewBox="0 0 100 100"
        preserveAspectRatio="none"
        x={`${index / props.heatmap.histograms.length * 100}%`}
        width={`${1 / props.heatmap.histograms.length * 100}%`}
        height="100%"
      >
        <HeatmapHistogram
          buckets={buckets}
          min={props.heatmap.min}
          max={props.heatmap.max}
        />
      </svg>
    );
  });
  let { min, max } = props.heatmap.histograms.reduce(
    (acc, h) => {
      let n = h[h.length - 1].c;
      if (n > acc.max) {
        acc.max = n;
      }
      if (n < acc.min) {
        acc.min = n;
      }
      return acc;
    },
    { min: Number.MAX_VALUE, max: -Number.MAX_VALUE }
  );
  let d = props.heatmap.histograms
    .map(buckets => {
      return buckets[buckets.length - 1].c;
    })
    .map((count, idx) => {
      let y = 100 / (max - min) * (count - min);
      let x =
        100.0 / props.heatmap.histograms.length * idx +
        50.0 / props.heatmap.histograms.length;
      return [x, y];
    })
    .reduce((acc, points) => {
      if (acc.length == 0) {
        acc = `M${points[0]} ${points[1]}`;
        return acc;
      }
      return acc + ` L ${points[0]} ${points[1]}`;
    }, "");
  return (
    <svg viewBox="0 0 100 100">
      <svg
        viewBox="0 0 100 100"
        width="100%"
        height="80%"
        preserveAspectRatio="none"
      >
        {histograms}
      </svg>
      <svg
        viewBox="0 0 100 100"
        width="100%"
        height="20%"
        y="80%"
        preserveAspectRatio="none"
      >
        <path d={d} fill="transparent" stroke="black" />{" "}
      </svg>
    </svg>
  );
};

export default Heatmap;
