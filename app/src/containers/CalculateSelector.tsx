import * as React from "react";
import { connect } from "reglaze";
import Select from "material-ui/Select";
import { MenuItem } from "material-ui/Menu";
import { State, Column } from "../state";
import { Action } from "../actions";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";

interface CalculateSelectorProps {
  select(column: Column): void;
  selected?: Column;
  columns?: Column[];
}

const CalculateSelector = (props: CalculateSelectorProps) => {
  if (!props.columns) {
    return <div />;
  }
  const options = props.columns.map((column: Column) => {
    return (
      <MenuItem key={column.field} value={column.field}>
        {column.field}
      </MenuItem>
    );
  });
  const value = props.selected ? props.selected.field : "";
  const lookup = props.columns.reduce((acc: any, column: Column): any => {
    return { ...acc, [column.field]: column };
  }, {});
  return (
    <Select
      value={value}
      onChange={event => {
        const column = lookup[event.target.value];
        if (!column) {
          return;
        }
        props.select(column);
      }}
    >
      {options}
    </Select>
  );
};

export default connect(
  "main",
  CalculateSelector,
  {
    columns: (state$: Observable<State>): Observable<Column[] | undefined> => {
      return state$.pipe(
        map((state: State): Column[] | undefined => {
          if (!state.service.selected) {
            return undefined;
          }
          return state.service.selected.columns;
        })
      );
    },
    selected: (state$: Observable<State>): Observable<Column | undefined> => {
      return state$.pipe(
        map((state: State): Column | undefined => {
          if (!state.service.selected) {
            return undefined;
          }
          if (!state.service.selected.calculate) {
            return undefined;
          }
          return state.service.selected.calculate;
        })
      );
    }
  },
  {
    select: (column: Column): Action => {
      return { kind: "SelectCalculateColumn", column };
    }
  }
);
