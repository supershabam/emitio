let component = ReasonReact.statelessComponent("Cell");

let schema =
  Chroma.Scale.domain(Chroma.Scale.scaleFromString("PuRd"), 1.0, 0.0);

let make =
    (
      ~intensity: float,
      ~top: float,
      ~left: float,
      ~bottom: float,
      ~right: float,
      _children,
    ) => {
  ...component,
  render: _self => {
    let fill = Chroma.Scale.hex(schema, intensity);
    <rect
      x=(string_of_float(left))
      y=(string_of_float(100.0 -. top))
      width=(string_of_float(right -. left))
      height=(string_of_float(top -. bottom))
      stroke="black"
      strokeWidth="0.1"
      fill
    />;
  },
};