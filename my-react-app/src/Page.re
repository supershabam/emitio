/* This is the basic component. */
let component = ReasonReact.statelessComponent("Page");

/* Your familiar handleClick from ReactJS. This mandatorily takes the payload,
   then the `self` record, which contains state (none here), `handle`, `reduce`
   and other utilities */
let handleClick = (_event, _self) => Js.log("clicked!");

/* `make` is the function that mandatorily takes `children` (if you want to use
   `JSX). `message` is a named argument, which simulates ReactJS props. Usage:

   `<Page message="hello" />`

   Which desugars to

   `ReasonReact.element(Page.make(~message="hello", [||]))` */
let make = (~message, ~color: Chroma.t, _children) => {
  ...component,
  render: self =>
    <div
      style=(
        ReactDOMRe.Style.make(~color=Chroma.hex(color), ~fontSize="68px", ())
      )
      onClick=(self.handle(handleClick))>
      (ReasonReact.stringToElement(message))
    </div>,
};