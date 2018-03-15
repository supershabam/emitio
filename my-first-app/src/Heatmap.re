type heatmap = {histograms: list(Histogram.histogram)};

let clip = (h: Histogram.histogram, start: float, end_: float) => {
  let next = {
    ...h,
    accuracy: h.accuracy *. (end_ -. start) /. (h.end_ -. h.start),
    start,
    end_,
  };
  next;
};

let inject = (wide: Histogram.histogram, narrow: Histogram.histogram) => {
  /* wide completely covers narrow */
  let wwidth = wide.end_ -. wide.start;
  let nwidth = narrow.end_ -. narrow.start;
  let wideacc = wide.accuracy *. nwidth /. wwidth;
  if (wideacc > narrow.accuracy) {
    [wide];
  } else {
    [
      clip(wide, wide.start, narrow.start),
      narrow,
      clip(wide, narrow.end_, wide.end_),
    ];
  };
};

let overlay = (bottom: Histogram.histogram, top: Histogram.histogram) =>
  switch (top, bottom) {
  | (t, b) when b.start < t.start => [clip(b, b.start, t.start), t]
  | _ => [top, clip(bottom, top.end_, bottom.end_)]
  };

let rec join = (a: Histogram.histogram, b: Histogram.histogram) =>
  switch (a, b) {
  | (a, b) when b.start < a.start => join(b, a) /* only handle one ordering where a.start <= b.start */
  | (a, b) when a.end_ <= b.start => [a, b] /* non-overlapping */
  | (a, b) when a.start <= b.start && a.end_ >= b.end_ => inject(a, b) /* a completely covers b */
  | (a, b) when b.start <= a.start && b.end_ >= a.end_ => inject(b, a) /* b completely covers a */
  | (a, b) when a.accuracy > b.accuracy => overlay(b, a) /* overlay a on top of b */
  | _ => overlay(a, b) /* overlay b on top of a */
  };