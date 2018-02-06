import * as React from "react";
import { connect } from "react-redux";
import Heatmap from "../components/Heatmap";

const s2p = state => {
  return {
    heatmap: state.heatmap
  };
};

export default connect(s2p)(Heatmap);
