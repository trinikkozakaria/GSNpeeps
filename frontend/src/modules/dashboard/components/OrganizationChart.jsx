import { useEffect, useRef, useState } from "react";

import { chartColorForIndex } from "../../../lib/chart-colors";
import { OrganizationEmployeeDetailModal } from "./OrganizationEmployeeDetailModal";

// Urutan visual mengikuti bagan organisasi resmi, bukan urutan alfabet dari API.
// Nama yang belum ada dalam referensi tetap ditampilkan setelah nama yang sudah dipetakan.
const memberOrder = [
  "Anthony Hamonangan Sihombing",
  "Rachmat Efendi",
  "Nadya Theresia Sihombing",
  "Mesianti Puspawardhany",
  "Dewa Pambudhi",
  "Beniqno Joe Prasetyo Siilitonga",
  "Bima sanjaya",
  "Arif Triantoro",
  "Rini Yulianto",
  "Refina Anastasya",
  "Nadia Cahya Rani",
  "Destaria Dwi Maryastuti",
  "Afrizal Ari Jotivian",
  "Ray Strainer, S.H.",
  "Arif Hidayat",
  "Saher Remal Agungta Ketaren",
  "Adisty Aulia Rosadi",
  "Ratu Fasya Dwinata Yusup",
  "Fajriani Oktavianur",
  "Naini Pujiati",
  "Pekik Satria Andika",
  "Yohanes Otto Hasudungan",
  "Afdah adi ugi",
  "Riza Perdana",
  "Muhammad Rifqy Prawira",
  "Wira Sanjaya",
  "Akhyar Muhataris Adhin",
  "Syifa Handayani",
  "Elis Syubban Al Fatih",
  "Aji Braja Yudha",
  "Aris Setiawan",
  "Adam Purwa Bagaskara",
  "Muhamad Rizaldi",
  "Rendi Oktobiwanto",
  "David Vio Ariyanda Putra",
  "Yosafat",
  "Rapidal Ajis",
  "Bimawan Zakaria",
  "Wawan Suryana",
  "Adhi Prihatmoko",
  "Darrel Hutomi",
  "Alfian Purnomo",
  "Tri Nikko Zakaria",
  "Benedicto Joko Ferdinand Silitonga",
  "Muhammad Al-Ghazali",
  "Agil Wahyudi Ariyanto",
  "Zainuddin oky wijaya",
  "Dimas Artha Prasetya",
  "Varrel Arya Yudhanto",
];

const memberOrderIndex = new Map(memberOrder.map((name, index) => [name, index]));

const sortMembers = (nodes) =>
  [...nodes].sort((left, right) => {
    const leftOrder = memberOrderIndex.get(left.nama) ?? Number.MAX_SAFE_INTEGER;
    const rightOrder = memberOrderIndex.get(right.nama) ?? Number.MAX_SAFE_INTEGER;
    return leftOrder - rightOrder || left.nama.localeCompare(right.nama, "id");
  });

// Relasi ini hanya memengaruhi garis koordinasi visual; struktur atasan_id tetap menjadi
// satu-satunya sumber garis pelaporan solid dan alur persetujuan.
const coordinationRelations = [
  { source: "Dewa Pambudhi", target: "Beniqno Joe Prasetyo Siilitonga" },
  { source: "Refina Anastasya", target: "Destaria Dwi Maryastuti", route: "source-elbow" },
  { source: "Nadia Cahya Rani", target: "Saher Remal Agungta Ketaren", route: "children-spine" },
  { source: "Afrizal Ari Jotivian", target: "Ray Strainer, S.H." },
  { source: "Naini Pujiati", target: "Pekik Satria Andika", route: "children-spine" },
  // Ughay belum tersedia di roster lokal, sehingga kartu pada posisi tersebut memakai Afdah.
  { source: ["Ughay", "Afdah adi ugi"], target: "Bimawan Zakaria" },
  { source: "Riza Perdana", target: "Bimawan Zakaria", route: "children-spine" },
];

const OrganizationCoordinationLines = ({ canvasRef }) => {
  const [paths, setPaths] = useState([]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return undefined;

    const updatePaths = () => {
      const canvasRect = canvas.getBoundingClientRect();
      const cards = [...canvas.querySelectorAll("[data-organization-employee]")];
      const findCard = (names) => {
        const candidates = Array.isArray(names) ? names : [names];
        return candidates
          .map((name) => cards.find((card) => card.dataset.organizationEmployee === name))
          .find(Boolean);
      };
      const branchTop = (card) => {
        const member = card.closest(".organization-tree__member");
        const branch = [...member.children].find((child) =>
          child.classList.contains("organization-tree__children"),
        );
        return branch?.getBoundingClientRect().top - canvasRect.top;
      };
      const nextPaths = coordinationRelations.flatMap(({ source: sourceName, target: targetName, route }) => {
        const source = findCard(sourceName);
        const target = findCard(targetName);
        if (!source || !target) return [];

        const sourceRect = source.getBoundingClientRect();
        const targetRect = target.getBoundingClientRect();
        const sourceIsLeft = sourceRect.left <= targetRect.left;
        const startX = (route === "source-elbow"
          ? sourceRect.left + sourceRect.width / 2
          : sourceIsLeft ? sourceRect.right : sourceRect.left) - canvasRect.left;
        const endX = (sourceIsLeft ? targetRect.left : targetRect.right) - canvasRect.left;
        const sourceIsAbove = sourceRect.top <= targetRect.top;
        const startY = (route === "source-elbow"
          ? sourceIsAbove ? sourceRect.bottom : sourceRect.top
          : sourceRect.top + sourceRect.height / 2) - canvasRect.top;
        const endY = targetRect.top + targetRect.height / 2 - canvasRect.top;

        if (route === "children-spine") {
          const sourceBranchY = branchTop(source);
          const targetBranchY = branchTop(target);
          if (sourceBranchY === undefined || targetBranchY === undefined) return [];
          const sourceCenterX = sourceRect.left + sourceRect.width / 2 - canvasRect.left;
          const targetCenterX = targetRect.left + targetRect.width / 2 - canvasRect.left;
          const sharedY = Math.max(sourceBranchY, targetBranchY);
          return [{
            key: `${source.dataset.organizationEmployee}-${target.dataset.organizationEmployee}`,
            d: `M ${sourceCenterX} ${sourceBranchY} V ${sharedY} H ${targetCenterX} V ${targetBranchY}`,
          }];
        }

        return [{
          key: `${source.dataset.organizationEmployee}-${target.dataset.organizationEmployee}`,
          d: route === "source-elbow"
            ? `M ${startX} ${startY} V ${endY} H ${endX}`
            : Math.abs(startY - endY) < 2
            ? `M ${startX} ${startY} H ${endX}`
            : `M ${startX} ${startY} V ${endY} H ${endX}`,
        }];
      });
      setPaths(nextPaths);
    };

    updatePaths();
    const frame = window.requestAnimationFrame(updatePaths);
    window.addEventListener("resize", updatePaths);
    const observer = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(updatePaths);
    observer?.observe(canvas);

    return () => {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("resize", updatePaths);
      observer?.disconnect();
    };
  }, [canvasRef]);

  return (
    <svg className="organization-tree__coordination" aria-hidden="true">
      {paths.map((path) => <path key={path.key} d={path.d} />)}
    </svg>
  );
};

/**
 * Org chart dirender sebagai daftar bersarang sehingga hubungan atasan-bawahan dapat
 * ditelusuri screen reader dan keyboard tanpa memerlukan grafik terpisah.
 */
const OrganizationMember = ({ node, onSelect, departmentColors }) => {
  const hasReports = node.bawahan.length > 0;
  const departmentColor = departmentColors.get(node.departemen);

  return (
    <li className="organization-tree__member">
      <div
        data-organization-employee={node.nama}
        className="organization-tree__card flex items-center justify-between gap-1 rounded-md border p-1 shadow-sm"
        style={departmentColor ? { "--organization-department-color": departmentColor } : undefined}
      >
        <div className="min-w-0">
          <p className="text-[9px] font-semibold leading-[1.15]">{node.nama}</p>
          <p className="organization-tree__subtitle mt-0.5 text-[7px] leading-[1.15]">
            {node.jabatan || "Jabatan belum ditetapkan"} ·{" "}
            {node.departemen || "Departemen belum ditetapkan"}
          </p>
        </div>
        <div className="flex shrink-0 items-center">
          <button
            type="button"
            className="organization-tree__detail py-0.5 text-[7px] font-semibold leading-tight underline-offset-2 hover:underline focus-visible:rounded-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-current"
            aria-label={`Lihat detail ${node.nama}`}
            onClick={() => onSelect(node)}
          >
            Lihat detail
          </button>
        </div>
      </div>
      {hasReports && (
        <OrganizationBranch
          nodes={node.bawahan}
          onSelect={onSelect}
          departmentColors={departmentColors}
        />
      )}
    </li>
  );
};

const OrganizationBranch = ({ nodes, onSelect, departmentColors, root = false }) => (
  <ul className={root ? "organization-tree__roots" : "organization-tree__children"}>
    {sortMembers(nodes).map((node) => (
      <OrganizationMember
        key={node.employee_id}
        node={node}
        onSelect={onSelect}
        departmentColors={departmentColors}
      />
    ))}
  </ul>
);

const countMembers = (nodes) =>
  nodes.reduce((total, node) => total + 1 + countMembers(node.bawahan), 0);

export const OrganizationChart = ({ nodes, departments = [] }) => {
  const [selectedEmployee, setSelectedEmployee] = useState(null);
  const canvasRef = useRef(null);
  const departmentColors = new Map(
    departments.map((department, index) => [department.nama, chartColorForIndex(index)]),
  );
  const ceo = nodes.find((node) => node.nama === "Anthony Hamonangan Sihombing");
  const officialNodes = ceo ? [ceo] : nodes;

  if (officialNodes.length === 0) {
    return (
      <p className="text-sm text-slate-500">
        Belum ada karyawan aktif untuk menyusun struktur organisasi pada periode ini.
      </p>
    );
  }

  return (
    <div>
      <p className="text-sm text-slate-500">
        {countMembers(officialNodes)} karyawan aktif dalam {officialNodes.length} jalur pelaporan teratas.
      </p>
      <div
        id="organization-chart-content"
        className="organization-tree mt-4 overflow-x-auto pb-3"
      >
        <div ref={canvasRef} className="organization-tree__canvas">
          <OrganizationBranch
            nodes={officialNodes}
            onSelect={setSelectedEmployee}
            departmentColors={departmentColors}
            root
          />
          <OrganizationCoordinationLines canvasRef={canvasRef} />
        </div>
      </div>
      <p className="organization-tree__legend mt-1 text-[9px] text-slate-500">
        <span aria-hidden="true" /> Garis putus-putus menunjukkan koordinasi.
      </p>
      {selectedEmployee && (
        <OrganizationEmployeeDetailModal
          employeeId={selectedEmployee.employee_id}
          employeeName={selectedEmployee.nama}
          onClose={() => setSelectedEmployee(null)}
        />
      )}
    </div>
  );
};
