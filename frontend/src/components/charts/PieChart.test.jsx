import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PieChart } from "./PieChart";

describe("PieChart", () => {
  it("shows the department detail when a segment is hovered or focused", () => {
    render(
      <PieChart
        title="Komposisi departemen"
        emptyMessage="Belum ada data"
        items={[
          { nama: "Engineering", jumlah: 15 },
          { nama: "Finance", jumlah: 5 },
        ]}
      />,
    );

    const engineering = screen.getByLabelText("Engineering: 15 karyawan, 75 persen");
    fireEvent.mouseEnter(engineering);
    expect(screen.getByRole("tooltip")).toHaveTextContent("Engineering: 15 karyawan (75%)");

    fireEvent.mouseLeave(engineering);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();

    fireEvent.focus(screen.getByLabelText("Finance: 5 karyawan, 25 persen"));
    expect(screen.getByRole("tooltip")).toHaveTextContent("Finance: 5 karyawan (25%)");
  });
});
