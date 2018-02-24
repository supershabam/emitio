import * as React from "react";
import { connect } from "react-redux";

import AutoComplete from "material-ui/AutoComplete";
import Slider from "material-ui/Slider";
import FloatingActionButton from "material-ui/FloatingActionButton";
import {
  Card,
  CardActions,
  CardTitle,
  CardHeader,
  CardText
} from "material-ui/Card";
import ContentAdd from "material-ui/svg-icons/content/add";
import ContentRemove from "material-ui/svg-icons/content/remove";
import RaisedButton from "material-ui/RaisedButton";
import {
  Table,
  TableBody,
  TableHeader,
  TableHeaderColumn,
  TableRow,
  TableRowColumn
} from "material-ui/Table";
import SelectField from "material-ui/SelectField";
import MenuItem from "material-ui/MenuItem";
import TextField from "material-ui/TextField";
import Rows from "./Rows";

const Service = props => {
  let acc = JSON.parse(props.acc);
  let process = acc.process;
  return (
    <div>
      <Card>
        <CardTitle title="Service" subtitle="Define a new service" />
        <CardText>
          <TextField defaultValue="prod api" floatingLabelText="Name" />
          <br />
          <TextField defaultValue="API Team" floatingLabelText="Owner" />
        </CardText>
      </Card>
      <br />
      <Card>
        <CardTitle
          title="Select"
          subtitle="Which emitio ingresses compose this service"
        />
        <CardText>
          <AutoComplete
            hintText="field"
            dataSource={["env", "region"]}
            openOnFocus={true}
            searchText={"origin.env"}
          />
          <SelectField floatingLabelText="match" value={1}>
            <MenuItem value={1} primaryText="equals" />
            <MenuItem value={2} primaryText="not equals" />
            <MenuItem value={3} primaryText="regex match" />
            <MenuItem value={4} primaryText="regex not match" />
          </SelectField>
          <AutoComplete
            hintText="value"
            dataSource={["prod", "stage"]}
            openOnFocus={true}
            searchText={"prod"}
          />
          <FloatingActionButton mini={true}>
            <ContentRemove />
          </FloatingActionButton>
          <hr />
          <AutoComplete
            hintText="field"
            dataSource={["env", "region"]}
            openOnFocus={true}
            searchText={"ingress.scheme"}
          />
          <SelectField floatingLabelText="match" value={1}>
            <MenuItem value={1} primaryText="equals" />
            <MenuItem value={2} primaryText="not equals" />
            <MenuItem value={3} primaryText="regex match" />
            <MenuItem value={4} primaryText="regex not match" />
          </SelectField>
          <AutoComplete
            hintText="value"
            dataSource={["prod", "stage"]}
            openOnFocus={true}
            searchText={"udp+syslog"}
          />
          <FloatingActionButton mini={true}>
            <ContentRemove />
          </FloatingActionButton>
          <br />
          <FloatingActionButton mini={true}>
            <ContentAdd />
          </FloatingActionButton>
          <br />
        </CardText>
      </Card>
      <br />
      <Card>
        <CardTitle title="Parse" subtitle="Enrich collected data" />
        <CardText>
          <SelectField floatingLabelText="parser" value={1}>
            <MenuItem value={1} primaryText="syslog" />
            <MenuItem value={3} primaryText="nginx" />
            <MenuItem value={2} primaryText="custom" />
          </SelectField>
          <FloatingActionButton mini={true}>
            <ContentRemove />
          </FloatingActionButton>
          <br />
          <SelectField floatingLabelText="parser" value={3}>
            <MenuItem value={1} primaryText="syslog" />
            <MenuItem value={3} primaryText="nginx" />
            <MenuItem value={2} primaryText="custom" />
          </SelectField>
          <FloatingActionButton mini={true}>
            <ContentRemove />
          </FloatingActionButton>
          <br />
          <FloatingActionButton mini={true}>
            <ContentAdd />
          </FloatingActionButton>
          <br />
        </CardText>
      </Card>
      <br />
      <Card>
        <CardTitle
          title="Filter"
          subtitle="Drop noisy or irrelevant messages"
        />
        <CardText>
          <AutoComplete
            hintText="field"
            dataSource={["env", "region"]}
            openOnFocus={true}
            searchText={"syslog.process"}
          />
          <SelectField floatingLabelText="match" value={2}>
            <MenuItem value={1} primaryText="equals" />
            <MenuItem value={2} primaryText="not equals" />
            <MenuItem value={3} primaryText="regex match" />
            <MenuItem value={4} primaryText="regex not match" />
          </SelectField>
          <AutoComplete
            hintText="value"
            dataSource={["sshd", "nginx"]}
            openOnFocus={true}
            searchText={process}
            onUpdateInput={props.setSelectedProcess}
          />
          <FloatingActionButton mini={true}>
            <ContentRemove />
          </FloatingActionButton>
          <hr />
          <FloatingActionButton mini={true}>
            <ContentAdd />
          </FloatingActionButton>
          <br />
        </CardText>
      </Card>
      <br />
      <Card>
        <CardTitle
          title="Dynamic Downsampling"
          subtitle="Maintain a budget with no code changes"
        />
        <CardText>
          <span>current writes per second: 253</span>
          <br />
          <br />
          <span>Transient writes per second: 1k</span>
          <Slider defaultValue={0.1} />
          <span>Burst writes per second: 5k</span>
          <Slider defaultValue={0.3} />
          <p>estimated cost: $3.50/month</p>
          <p>maximum cost: $45/month</p>
        </CardText>
      </Card>

      <br />
      <Card>
        <RaisedButton
          primary={true}
          onClick={() =>
            props.readNode(props.node, props.javascript, props.acc)
          }
        >
          Sample
        </RaisedButton>
        <Rows />
      </Card>
    </div>
  );
};

const mapStateToProps = state => {
  return {
    acc: state.acc,
    node: state.nodes[0],
    javascript: state.value
  };
};

const mapDispatchToProps = dispatch => {
  return {
    setSelectedProcess: process =>
      dispatch({
        type: "SET_ACC",
        acc: `{"process":"${process}"}`
      }),
    readNode: (node, javascript, acc) =>
      dispatch({
        type: "READ_NODE",
        request: {
          uri: "syslog+udp://127.0.0.1:514",
          start: 0,
          end: Number.MAX_SAFE_INTEGER,
          node: node,
          accumulator: acc,
          input_limit: 1000,
          output_limit: 1000,
          duration_limit: 15,
          javascript: javascript
        }
      })
  };
};

export default connect(mapStateToProps, mapDispatchToProps)(Service);
