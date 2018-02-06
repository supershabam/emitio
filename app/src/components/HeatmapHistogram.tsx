import * as React from "react";

const HeatmapHistogram = props => {
  // linear where 5% height is min and 95% of height is max
  let yFn = x => 90.0 / (props.max - props.min) * (x - props.min) + 5;
  let total = props.buckets[props.buckets.length - 1].c;
  let buckets = props.buckets.reduce(
    (acc, bucket) => {
      let le = parseFloat(bucket.le);
      let diff = bucket.c - acc.lastCount;
      let y = 100;
      if (isFinite(le)) {
        y = yFn(le);
      }
      let dy = y - acc.lastY;
      let width = y - acc.lastY;
      let intensity = diff / total * width;
      let el = (
        <rect
          key={y}
          width="100%"
          y={`${100 - y}%`}
          height={`${dy}%`}
          fill="red"
          stroke="black"
          opacity={`${intensity}`}
        />
      );
      return {
        lastY: y,
        lastCount: bucket.c,
        el: acc.el.concat(el)
      };
    },
    { lastY: 0, lastCount: 0, el: [] }
  );
  return (
    <svg viewBox="0 0 100 100" preserveAspectRatio="none">
      {buckets.el}
    </svg>
  );
};

export default HeatmapHistogram;
