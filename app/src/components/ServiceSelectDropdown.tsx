import * as React from "react";
import Select from "material-ui/Select";

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
      <option key={service} value={service}>
        {service}
      </option>
    );
  });
  console.log(selected, value);
  return (
    <Select
      value={value}
      onClick={(event: any) => {
        select(event.target.value);
      }}
    >
      {options}
    </Select>
  );
};

export default ServiceSelectDropdown;
