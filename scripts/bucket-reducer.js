function transform(acc, lines) {
  let a = JSON.parse(acc);
  a.buckets = a.buckets || [
    { le: 0.1, c: 0 },
    { le: 0.2, c: 0 },
    { le: 0.4, c: 0 }
  ];
  let out = lines
    .map(line => JSON.parse(line))
    .filter(line => line.r.match(/histtest/))
    .map(line => {
      let re = /response_time=([0-9\.]+)/;
      let match = line.r.match(re);
      if (match) {
        let responseTime = parseFloat(match[1]);
        a.buckets = a.buckets.map(bucket => {
          return responseTime < bucket.le
            ? Object.assign({}, bucket, { c: bucket.c + 1 })
            : bucket;
        });
        line = Object.assign({}, line, { r: parseFloat(match[1]) });
      }
      return line;
    })
    .map(line => JSON.stringify(line));
  a.count = (a.count || 0) + lines.length;
  return [JSON.stringify(a), out];
}
