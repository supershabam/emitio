// Generated by BUCKLESCRIPT VERSION 2.2.2, PLEASE EDIT WITH CARE
'use strict';


function clip(h, start, end_) {
  return /* record */[
          /* buckets */h[/* buckets */0],
          /* start */start,
          /* end_ */end_,
          /* accuracy */h[/* accuracy */3] * (end_ - start) / (h[/* end_ */2] - h[/* start */1])
        ];
}

function inject(wide, narrow) {
  var wwidth = wide[/* end_ */2] - wide[/* start */1];
  var nwidth = narrow[/* end_ */2] - narrow[/* start */1];
  var wideacc = wide[/* accuracy */3] * nwidth / wwidth;
  if (wideacc > narrow[/* accuracy */3]) {
    return /* :: */[
            wide,
            /* [] */0
          ];
  } else {
    return /* :: */[
            clip(wide, wide[/* start */1], narrow[/* start */1]),
            /* :: */[
              narrow,
              /* :: */[
                clip(wide, narrow[/* end_ */2], wide[/* end_ */2]),
                /* [] */0
              ]
            ]
          ];
  }
}

function overlay(bottom, top) {
  if (bottom[/* start */1] < top[/* start */1]) {
    return /* :: */[
            clip(bottom, bottom[/* start */1], top[/* start */1]),
            /* :: */[
              top,
              /* [] */0
            ]
          ];
  } else {
    return /* :: */[
            top,
            /* :: */[
              clip(bottom, top[/* end_ */2], bottom[/* end_ */2]),
              /* [] */0
            ]
          ];
  }
}

function join(_a, _b) {
  while(true) {
    var b = _b;
    var a = _a;
    if (b[/* start */1] < a[/* start */1]) {
      _b = a;
      _a = b;
      continue ;
      
    } else if (a[/* end_ */2] <= b[/* start */1]) {
      return /* :: */[
              a,
              /* :: */[
                b,
                /* [] */0
              ]
            ];
    } else if (a[/* start */1] <= b[/* start */1] && a[/* end_ */2] >= b[/* end_ */2]) {
      return inject(a, b);
    } else if (b[/* start */1] <= a[/* start */1] && b[/* end_ */2] >= a[/* end_ */2]) {
      return inject(b, a);
    } else if (a[/* accuracy */3] > b[/* accuracy */3]) {
      return overlay(b, a);
    } else {
      return overlay(a, b);
    }
  };
}

exports.clip = clip;
exports.inject = inject;
exports.overlay = overlay;
exports.join = join;
/* No side effect */