import * as React from "react";
import Select from "material-ui/Select";
import { MenuItem } from "material-ui/Menu";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";
import { State } from "../state";
import { Action } from "../actions";
import { connect } from "reglaze";

interface ServiceSelectDropdownProps {
  services: string[];
  selected?: string;
  select(selection: string): void;
}

const ServiceSelectDropdown = (props: ServiceSelectDropdownProps) => {
  const { select, selected, services } = props;
  const value = selected || "";
  const options = (services || []).map((service: string) => {
    return (
      <MenuItem key={service} value={service}>
        {service}
      </MenuItem>
    );
  });
  return (
    <Select
      value={value}
      onClick={(event: any) => {
        select(event.target.value);
      }}
    >
      <MenuItem value="">
        <em>None</em>
      </MenuItem>
      {options}
    </Select>
  );
};

export default connect(
  "main",
  ServiceSelectDropdown,
  {
    selected: ($state: Observable<State>): Observable<string> => {
      return $state.pipe(
        map((state: State): string => {
          if (state.service.selected) {
            return state.service.selected;
          }
          return "";
        })
      );
    },
    services: ($state: Observable<State>): Observable<string[]> => {
      return $state.pipe(
        map((state: State): string[] => {
          return state.service.services;
        })
      );
    }
  },
  {
    select: (selected: string): Action => {
      return { kind: "SelectService", selected };
    }
  }
);
