import * as React from "react";
import { connect } from "react-redux";

const Rows = props => {
  let rows = props.value.map(row => {
    return (
      <tr key={row.s}>
        <td>{row.s}</td>
        <td>{row.a}</td>
        <td>{row.r}</td>
      </tr>
    );
  });
  return <table>{rows}</table>;
};

export default connect(state => {
  return {
    total: state.rows.length,
    value: state.rows.slice(0, 100)
  };
})(Rows);
