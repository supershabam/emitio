import * as React from "react";
import * as CodeMirror from "react-codemirror";
import "codemirror/mode/javascript/javascript";
import { connect } from "react-redux";

const NodeCount = props => {
  return <span>active nodes={props.count}</span>;
};

export default connect(state => {
  return {
    count: state.nodes.length
  };
})(NodeCount);
