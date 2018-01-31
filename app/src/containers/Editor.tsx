import * as React from "react";
import * as CodeMirror from "react-codemirror";
import "codemirror/mode/javascript/javascript";
import { connect } from "react-redux";

const Editor = props => {
  return (
    <CodeMirror
      mode="javascript"
      value={props.value}
      onChange={newValue => props.onChange(newValue)}
    />
  );
};

const mapStateToProps = state => {
  return {
    value: state.value
  };
};

export default connect(mapStateToProps, dispatch => {
  return {
    onChange: newValue => dispatch({ type: "EDITOR_CHANGE", value: newValue })
  };
})(Editor);
