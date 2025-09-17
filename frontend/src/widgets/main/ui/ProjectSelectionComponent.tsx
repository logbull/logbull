import { Button, Input } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';

import { type ProjectResponse } from '../../../entity/projects';

interface Props {
  projects: ProjectResponse[];
  selectedProject?: ProjectResponse;
  onCreateProject: () => void;
  onProjectSelect: (project: ProjectResponse) => void;
}

export const ProjectSelectionComponent = ({
  projects,
  selectedProject,
  onCreateProject,
  onProjectSelect,
}: Props) => {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  const filteredProjects = useMemo(() => {
    if (!searchValue.trim()) return projects;
    const searchTerm = searchValue.toLowerCase();
    return projects.filter((project) => project.name.toLowerCase().includes(searchTerm));
  }, [projects, searchValue]);

  const openProject = (project: ProjectResponse) => {
    setIsDropdownOpen(false);
    setSearchValue('');
    onProjectSelect?.(project);
  };

  // Handle click outside dropdown
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
        setSearchValue('');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  if (projects.length === 0) {
    return (
      <Button
        type="primary"
        onClick={onCreateProject}
        className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
      >
        Create project
      </Button>
    );
  }

  return (
    <div className="my-1 w-[250px] select-none" ref={dropdownRef}>
      <div className="mb-1 text-xs text-gray-400" style={{ lineHeight: 0.7 }}>
        Selected project
      </div>

      <div className="relative">
        {/* Dropdown Trigger */}
        <div
          className="cursor-pointer rounded bg-gray-100 p-1 px-2 hover:bg-gray-200"
          onClick={() => setIsDropdownOpen(!isDropdownOpen)}
        >
          <div className="flex items-center justify-between text-sm">
            <div className="max-w-[250px] truncate">
              {selectedProject?.name || 'Select a project'}
            </div>
            <img
              src="/icons/menu/arrow-down-gray.svg"
              alt="arrow-down"
              className={`ml-1 transition-transform duration-200 ${isDropdownOpen ? 'rotate-180' : ''}`}
              width={15}
              height={15}
            />
          </div>
        </div>

        {/* Dropdown Menu */}
        {isDropdownOpen && (
          <div className="absolute top-full left-0 z-50 mt-1 min-w-full rounded-md border border-gray-200 bg-white shadow-lg">
            {/* Search Input */}
            <div className="border-b border-gray-100 p-2">
              <Input
                placeholder="Search projects..."
                value={searchValue}
                onChange={(e) => setSearchValue(e.target.value)}
                className="border-0 shadow-none"
                autoFocus
              />
            </div>

            {/* Project List */}
            <div className="max-h-[400px] overflow-y-auto">
              {filteredProjects.map((project) => (
                <div
                  key={project.id}
                  className="max-w-[250px] cursor-pointer truncate px-3 py-2 text-sm hover:bg-gray-50"
                  onClick={() => openProject(project)}
                >
                  {project.name}
                </div>
              ))}

              {filteredProjects.length === 0 && searchValue && (
                <div className="px-3 py-2 text-sm text-gray-500">No projects found</div>
              )}
            </div>

            {/* Create New Project Button - Fixed at bottom */}
            <div className="border-t border-gray-100">
              <div
                className="cursor-pointer px-3 py-2 text-sm text-emerald-600 hover:bg-gray-50 hover:text-emerald-700"
                onClick={() => {
                  onCreateProject();
                  setIsDropdownOpen(false);
                  setSearchValue('');
                }}
              >
                + Create new project
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
