type t;

[@bs.val] [@bs.module "chroma-js"] external chroma : string => t = "default";

[@bs.val] [@bs.module "chroma-js"] external random : unit => t = "random";

[@bs.send] external hex : t => string = "";

module Scale = {
  type t;
  [@bs.val] [@bs.module "chroma-js"]
  external scaleFromString : string => t = "scale";
  let domain: (t, float, float) => t = [%raw
    {|
    function(scale, a, b) {
        return scale.domain([a,b]);
    }
  |}
  ];
  let hex: (t, float) => string = [%raw
    {|
    function(scale, val) {
        return scale(val);
    }
  |}
  ];
};