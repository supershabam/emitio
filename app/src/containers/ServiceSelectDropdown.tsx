import * as React from "react";
import Select from "material-ui/Select";
import { MenuItem } from "material-ui/Menu";
import { Observable } from "rxjs";
import { map } from "rxjs/operators";
import { State, Service } from "../state";
import { Action } from "../actions";
import { connect } from "reglaze";

interface ServiceSelectDropdownProps {
  services: Service[];
  selected?: Service;
  select(selection: string): void;
}

const ServiceSelectDropdown = (props: ServiceSelectDropdownProps) => {
  const { select, selected, services } = props;
  const value = selected ? selected.id : "";
  const options = (services || []).map((service: Service) => {
    return (
      <MenuItem key={service.id} value={service.id}>
        {service.name}
      </MenuItem>
    );
  });
  const lookup = (services || []).reduce((acc: any, service: Service): any => {
    return { ...acc, [service.id]: service };
  }, {});
  return (
    <Select
      value={value}
      onClick={(event: any) => {
        if (event.target.value) {
          select(lookup[event.target.value]);
        }
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
    selected: ($state: Observable<State>): Observable<Service> => {
      return $state.pipe(
        map((state: State): Service => {
          if (state.service.selected) {
            return state.service.selected.service;
          }
          return { id: "", name: "" };
        })
      );
    },
    services: ($state: Observable<State>): Observable<Service[]> => {
      return $state.pipe(
        map((state: State): Service[] => {
          return state.service.services;
        })
      );
    }
  },
  {
    select: (service: Service): Action => {
      return { kind: "SelectService", service };
    }
  }
);
