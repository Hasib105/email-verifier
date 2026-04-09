import { useCallback, useEffect, useState } from 'react';
import { Check, Pencil, Plus, Trash2, X } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api';
import type { EmailTemplate, EmailTemplateCreateRequest } from '../types';

const inputClass = 'mt-1 block w-full border border-slate-300 px-3 py-2 text-sm text-slate-950 outline-none focus:border-slate-950 focus:ring-1 focus:ring-slate-950';

export function Templates() {
  const { config } = useAuth();
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const [formData, setFormData] = useState<EmailTemplateCreateRequest>({
    name: '',
    subject_template: '',
    body_template: '',
    active: true,
  });

  const loadTemplates = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.listEmailTemplates(config);
      setTemplates(response.items || []);
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load templates');
    } finally {
      setLoading(false);
    }
  }, [config]);

  useEffect(() => {
    void loadTemplates();
  }, [loadTemplates]);

  const resetForm = () => {
    setFormData({
      name: '',
      subject_template: '',
      body_template: '',
      active: true,
    });
    setEditingId(null);
    setShowForm(false);
  };

  const handleEdit = (template: EmailTemplate) => {
    setFormData({
      name: template.name,
      subject_template: template.subject_template,
      body_template: template.body_template,
      active: template.active,
    });
    setEditingId(template.id);
    setShowForm(true);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this template?')) return;

    try {
      await api.deleteEmailTemplate(config, id);
      await loadTemplates();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete template');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError('');

    try {
      if (editingId) {
        await api.updateEmailTemplate(config, editingId, formData);
      } else {
        await api.createEmailTemplate(config, formData);
      }
      resetForm();
      await loadTemplates();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save template');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mx-auto max-w-[1280px] space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-slate-500">Probe content</p>
          <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-950 sm:text-3xl">Email Templates</h1>
          <p className="mt-1 text-sm text-slate-500">Manage reusable messages for verification probes.</p>
        </div>
        {!showForm && (
          <button
            className="inline-flex h-9 items-center justify-center gap-2 bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800"
            onClick={() => setShowForm(true)}
            type="button"
          >
            <Plus className="h-4 w-4" />
            New Template
          </button>
        )}
      </div>

      {error && (
        <div className="border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {showForm && (
        <section className="border border-slate-200 bg-white">
          <div className="flex items-center justify-between border-b border-slate-200 px-5 py-4">
            <h2 className="text-base font-bold text-slate-950">{editingId ? 'Edit Template' : 'Create Template'}</h2>
            <button className="p-2 text-slate-500 hover:bg-slate-100" onClick={resetForm} type="button">
              <X className="h-4 w-4" />
            </button>
          </div>

          <form className="space-y-5 p-5" onSubmit={handleSubmit}>
            <div>
              <label className="block text-sm font-semibold text-slate-700">Template Name</label>
              <input className={inputClass} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder="Welcome Verification" required type="text" value={formData.name} />
            </div>
            <div>
              <label className="block text-sm font-semibold text-slate-700">Subject Line</label>
              <input className={inputClass} onChange={(e) => setFormData({ ...formData, subject_template: e.target.value })} placeholder="Action Required - Verify your email" required type="text" value={formData.subject_template} />
            </div>
            <div>
              <label className="block text-sm font-semibold text-slate-700">Body Template</label>
              <textarea className={`${inputClass} min-h-36`} onChange={(e) => setFormData({ ...formData, body_template: e.target.value })} placeholder="Hi, please verify your email by replying to this message..." required value={formData.body_template} />
              <p className="mt-2 text-xs text-slate-500">Use {'{{token}}'} for the verification token and {'{{email}}'} for the recipient email.</p>
            </div>
            <label className="inline-flex items-center gap-2 text-sm font-medium text-slate-700">
              <input checked={formData.active} className="h-4 w-4 border-slate-300" onChange={(e) => setFormData({ ...formData, active: e.target.checked })} type="checkbox" />
              Active template
            </label>
            <div className="flex gap-3 border-t border-slate-200 pt-5">
              <button className="h-9 bg-slate-950 px-4 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-50" disabled={saving} type="submit">
                {saving ? 'Saving...' : editingId ? 'Update' : 'Create'}
              </button>
              <button className="h-9 border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 hover:bg-slate-50" onClick={resetForm} type="button">
                Cancel
              </button>
            </div>
          </form>
        </section>
      )}

      {loading ? (
        <div className="p-10 text-center">
          <div className="mx-auto h-6 w-6 animate-spin rounded-full border-2 border-slate-200 border-t-slate-950" />
        </div>
      ) : templates.length === 0 ? (
        <section className="border border-slate-200 bg-white p-10 text-center text-sm text-slate-500">
          No templates created yet.
        </section>
      ) : (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {templates.map((template) => (
            <article className="border border-slate-200 bg-white p-5" key={template.id}>
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="truncate text-base font-bold text-slate-950">{template.name}</h2>
                    {template.active && (
                      <span className="inline-flex items-center gap-1 bg-emerald-50 px-2 py-1 text-xs font-semibold text-emerald-700">
                        <Check className="h-3 w-3" />
                        Active
                      </span>
                    )}
                  </div>
                  <p className="mt-2 text-sm text-slate-500">Subject: {template.subject_template}</p>
                </div>
                <div className="flex shrink-0 gap-1">
                  <button className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-slate-100 hover:text-slate-950" onClick={() => handleEdit(template)} type="button">
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button className="inline-flex h-8 w-8 items-center justify-center text-slate-500 hover:bg-red-50 hover:text-red-600" onClick={() => handleDelete(template.id)} type="button">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
              <div className="mt-5 max-h-40 overflow-auto border border-slate-200 bg-slate-50 p-4 text-sm leading-6 text-slate-600">
                {template.body_template}
              </div>
            </article>
          ))}
        </div>
      )}
    </div>
  );
}
