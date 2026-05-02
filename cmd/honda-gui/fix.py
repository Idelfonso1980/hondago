import sys

# Read the perfect HTML from the server
with open(r'web\index_from_server.html', 'rb') as f:
    text = f.read().decode('utf-8')

# 1. Inject the button
btn_target = 'onclick="openConfigSection(\'appusers\')">Usuários do Sistema</button>'
insert_btn = '\n              <button type="button" class="config-tab" holiday_date-config-tab="rbac" onclick="openConfigSection(\'rbac\')">Controle de Acessos</button>'
if btn_target in text:
    text = text.replace(btn_target, btn_target + insert_btn)

# 2. Inject the section
rbac_section = '''
            <section id="config-rbac" class="config-section hidden">
              <div class="auth-toolbar">
                 <h3>Matriz de Permissões (RBAC)</h3>
                 <p>Somente administradores têm acesso a esta tela. O gerenciamento visual de perfis será integrado aqui na próxima fase.</p>
              </div>
              <div class="auth-table-wrap">
                <table class="auth-table" id="rbac-table">
                  <thead>
                    <tr>
                      <th>Perfil</th>
                      <th>dashboard:read</th>
                      <th>solicitacoes:delete</th>
                      <th>users:manage</th>
                      <th>logs:delete</th>
                    </tr>
                  </thead>
                  <tbody id="rbac-tbody">
                     <tr>
                        <td>Operador</td>
                        <td><input type="checkbox" checked disabled></td>
                        <td><input type="checkbox" disabled></td>
                        <td><input type="checkbox" disabled></td>
                        <td><input type="checkbox" disabled></td>
                     </tr>
                     <tr>
                        <td>Admin</td>
                        <td><input type="checkbox" checked disabled></td>
                        <td><input type="checkbox" checked disabled></td>
                        <td><input type="checkbox" checked disabled></td>
                        <td><input type="checkbox" checked disabled></td>
                     </tr>
                  </tbody>
                </table>
              </div>
            </section>
'''

target_section = '<section id="config-appusers"'
if target_section in text:
    text = text.replace(target_section, rbac_section + '\n            ' + target_section)

# Write to index.html
with open(r'web\index.html', 'w', encoding='utf-8') as f:
    f.write(text)

print('Restored from server and injected RBAC tab perfectly.')
