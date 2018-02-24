import * as React from "react";
import { connect } from "react-redux";
import { Tabs, Tab } from "material-ui/Tabs";
import RaisedButton from "material-ui/RaisedButton";
import {
  Card,
  CardActions,
  CardHeader,
  CardMedia,
  CardTitle,
  CardText
} from "material-ui/Card";
import Editor from "./Editor";
import Rows from "./Rows";
import NodeCount from "./NodeCount";
import Heatmap from "./Heatmap";

const styles = {
  headline: {
    fontSize: 24,
    paddingTop: 16,
    marginBottom: 12,
    fontWeight: 400
  }
};

const App = props => {
  return (
    <Tabs value={props.tab} onChange={props.changeTab}>
      <Tab label="Reducer Experiment" value="reducer">
        <div>
          <NodeCount />
          <Card>
            <Editor />
          </Card>
          <br />
          <RaisedButton
            primary={true}
            onClick={() => props.readNode(props.node, props.javascript)}
          >
            Submit
          </RaisedButton>
          <hr />
          <Rows />
        </div>
      </Tab>
      <Tab label="Heatmap Experiment" value="heatmap">
        <div>
          <Card>
            <CardTitle title="Heatmap Generator" />
            <CardText>
              <Heatmap />
            </CardText>
            <CardActions>
              <RaisedButton primary={true} onClick={props.fetchHeatmap}>
                Regenerate
              </RaisedButton>
            </CardActions>
          </Card>
        </div>
      </Tab>
    </Tabs>
  );
};

const mapStateToProps = state => {
  return {
    node: state.nodes[0],
    javascript: state.value,
    tab: state.tab
  };
};

const mapDispatchToProps = dispatch => {
  return {
    changeTab: value => dispatch({ type: "CHANGE_TAB", value: value }),
    fetchHeatmap: () => dispatch({ type: "FETCH_HEATMAP" }),
    readNode: (node, javascript) =>
      dispatch({
        type: "READ_NODE",
        request: {
          uri: "syslog+udp://127.0.0.1:514",
          start: 0,
          end: Number.MAX_SAFE_INTEGER,
          node: node,
          accumulator: `{"process":"nginx"}`,
          input_limit: 1000,
          output_limit: 1000,
          duration_limit: 15,
          javascript: javascript
        }
      })
  };
};

export default connect(mapStateToProps, mapDispatchToProps)(App);
