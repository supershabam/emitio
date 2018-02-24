import * as React from "react";
import * as CodeMirror from "react-codemirror";
import "codemirror/mode/javascript/javascript";
import { connect } from "react-redux";

const Rows = props => {
  let rows = props.value.map(row => {
    return (
      <tr>
        <td>{row}</td>
      </tr>
    );
  });
  return (
    <div>
      <span>
        displaying {props.value.length} of {props.total}
      </span>
      <hr />
      <table>
        <tbody>{rows}</tbody>
      </table>
    </div>
  );
};

export default connect(state => {
  return {
    accumulator: state.lastAcc,
    total: state.rows.length,
    value: state.rows.slice(0, 100)
  };
})(Rows);
