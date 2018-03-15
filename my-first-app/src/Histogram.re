type histogram = {
  buckets: list(Bucket.bucket),
  start: float,
  end_: float,
  accuracy: float,
};

let bmin = (b: list(Bucket.bucket)) =>
  switch (b) {
  | [] => max_float
  | [head, ..._rest] => head.lte
  };

let rec bmax = (b: list(Bucket.bucket)) =>
  switch (b) {
  | [] => min_float
  | [last] => last.lte
  | [_head, ...rest] => bmax(rest)
  };

let hmin = (h: histogram) => bmin(h.buckets);

let hmax = (h: histogram) => bmax(h.buckets);